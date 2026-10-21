package health_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NorthDice/fugu-ocean/internal/health"
)

type fakePinger struct {
	err   error
	delay time.Duration
	// deadline captures ctx.Deadline() as observed inside Ping, so tests can
	// assert the service applies its own per-check timeout.
	deadline func(ctx context.Context)
}

func (f fakePinger) Ping(ctx context.Context) error {
	if f.deadline != nil {
		f.deadline(ctx)
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func TestService_Check_AllOK(t *testing.T) {
	svc := health.NewService(time.Second,
		health.Check{Name: "postgres", Pinger: fakePinger{}},
		health.Check{Name: "minio", Pinger: fakePinger{}},
		health.Check{Name: "meilisearch", Pinger: fakePinger{}},
	)

	report := svc.Check(context.Background())

	if report.Status != health.StatusOK {
		t.Fatalf("report.Status = %q, want %q", report.Status, health.StatusOK)
	}
	if len(report.Checks) != 3 {
		t.Fatalf("len(report.Checks) = %d, want 3", len(report.Checks))
	}
	for _, c := range report.Checks {
		if c.Status != health.StatusOK {
			t.Errorf("check %q status = %q, want %q", c.Name, c.Status, health.StatusOK)
		}
		if c.Error != "" {
			t.Errorf("check %q error = %q, want empty", c.Name, c.Error)
		}
	}
}

func TestService_Check_OneFails(t *testing.T) {
	boom := errors.New("boom")
	svc := health.NewService(time.Second,
		health.Check{Name: "postgres", Pinger: fakePinger{}},
		health.Check{Name: "minio", Pinger: fakePinger{err: boom}},
		health.Check{Name: "meilisearch", Pinger: fakePinger{}},
	)

	report := svc.Check(context.Background())

	if report.Status != health.StatusError {
		t.Fatalf("report.Status = %q, want %q", report.Status, health.StatusError)
	}

	for _, c := range report.Checks {
		switch c.Name {
		case "minio":
			if c.Status != health.StatusError {
				t.Errorf("minio status = %q, want %q", c.Status, health.StatusError)
			}
			if c.Error != boom.Error() {
				t.Errorf("minio error = %q, want %q", c.Error, boom.Error())
			}
		case "postgres", "meilisearch":
			if c.Status != health.StatusOK {
				t.Errorf("%s status = %q, want %q", c.Name, c.Status, health.StatusOK)
			}
		default:
			t.Errorf("unexpected check name %q", c.Name)
		}
	}
}

func TestService_Check_PreservesCheckOrder(t *testing.T) {
	// The slower check is declared first; the result slice must still come
	// back in declaration order rather than completion order.
	svc := health.NewService(time.Second,
		health.Check{Name: "slow", Pinger: fakePinger{delay: 30 * time.Millisecond}},
		health.Check{Name: "fast", Pinger: fakePinger{}},
	)

	report := svc.Check(context.Background())

	if len(report.Checks) != 2 {
		t.Fatalf("len(report.Checks) = %d, want 2", len(report.Checks))
	}
	if report.Checks[0].Name != "slow" || report.Checks[1].Name != "fast" {
		t.Fatalf("report.Checks = %+v, want order [slow, fast]", report.Checks)
	}
}

func TestService_Check_AppliesPerCheckTimeout(t *testing.T) {
	const timeout = 20 * time.Millisecond

	start := time.Now()
	var observedDeadline time.Time
	var hadDeadline bool

	svc := health.NewService(timeout,
		health.Check{Name: "slow", Pinger: fakePinger{
			deadline: func(ctx context.Context) {
				observedDeadline, hadDeadline = ctx.Deadline()
			},
			delay: time.Second, // much longer than the timeout
		}},
	)

	report := svc.Check(context.Background())

	if !hadDeadline {
		t.Fatal("Ping() was not given a context deadline")
	}
	if got := observedDeadline.Sub(start); got > timeout+50*time.Millisecond {
		t.Fatalf("deadline too far out: got %s, want ~%s", got, timeout)
	}

	if report.Checks[0].Status != health.StatusError {
		t.Fatalf("status = %q, want %q (should time out)", report.Checks[0].Status, health.StatusError)
	}
}

func TestService_Check_NoChecksIsOK(t *testing.T) {
	svc := health.NewService(time.Second)

	report := svc.Check(context.Background())

	if report.Status != health.StatusOK {
		t.Fatalf("report.Status = %q, want %q", report.Status, health.StatusOK)
	}
	if len(report.Checks) != 0 {
		t.Fatalf("len(report.Checks) = %d, want 0", len(report.Checks))
	}
}
