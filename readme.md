Dynamic :

```yaml
http:
  # https://doc.traefik.io/traefik/middlewares/http/overview/
  middlewares:
    rtb:
      plugin:
        rtb_static:
          cookieName: "pod-id"
          responseTimeHeaderName: "Tm"
          responseTimeLimitMs: "2"
```