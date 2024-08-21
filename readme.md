Dynamic :

```yaml
http:
  # https://doc.traefik.io/traefik/middlewares/http/overview/
  middlewares:
    rtb:
      plugin:
        rtb_static:
          responseTimeHeaderName: "Tm"
          responseTimeLimitMs: "80"
          cookieSetHeaderValue: "invalidated"

          logStartup: true
          logSetCookie: true
          logLimitNotReached: true
          logHeaderNotFound: true
```