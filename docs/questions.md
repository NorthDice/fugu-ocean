# Questions Log

A running log of questions that came up while planning/building this project, with answers.

---

## Q1: What kinds of API versioning exist besides `/v1`?

**A:**

1. **URI path versioning** — `/v1/files`. The version is part of the URL itself.
   - Pros: most explicit and discoverable, trivial to route (reverse proxy / router can split
     traffic by path prefix), easy to test with curl, easy to run two versions side by side.
   - Cons: technically the same resource gets a different URL per version, which is debatable
     from a strict REST purity standpoint. In practice nobody cares.
   - This is the most common approach for REST APIs and what most frameworks default to.

2. **Query parameter versioning** — `/files?version=1`.
   - Pros: URL path stays "clean", trivial to add a default.
   - Cons: easy to forget/omit, less visible in logs and API docs, awkward to route at the
     reverse-proxy level (most proxies route by path, not query string).

3. **Custom header versioning** — `X-API-Version: 1`.
   - Pros: URL and resource identity stay stable across versions.
   - Cons: invisible when just looking at a URL, can't be bookmarked/shared/tested by pasting
     a link in a browser, every client call needs an extra header — easy to forget in a new client.

4. **Accept header / content negotiation** — `Accept: application/vnd.fuguocean.v1+json`.
   - Pros: closest to "proper" REST per the HTTP spec (versioning the representation, not the resource).
   - Cons: most complex for clients to use correctly, harder to debug/test manually, rarely worth
     the friction outside of API-purist teams.

5. **Subdomain/hostname versioning** — `v1.api.example.com`.
   - Pros: clean infrastructure-level separation per version (can even be different deployments).
   - Cons: needs DNS/infra setup per version, overkill unless you're running major versions on
     genuinely separate backends.

**Decision for this project:** use URI path versioning (`/v1/...`). It's the simplest, most visible,
and cheapest to add now vs. later — worth baking into the router from Module 0 instead of retrofitting.
