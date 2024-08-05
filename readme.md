Dynamic :

```yaml
http:
  # https://doc.traefik.io/traefik/middlewares/http/overview/
  middlewares:
    rtb:
      plugin:
        rtb_static:
          cookie_name: "pod-id"
          response_time_limit_ms: 50
          response_time_header_name: "Tm"
```