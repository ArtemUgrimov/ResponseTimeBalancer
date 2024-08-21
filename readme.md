# Configuration examples

## Local file provider

### Static config
```yaml
experimental:
  localPlugins:
    rtb_static:
      moduleName: "github.com/ArtemUgrimov/ResponseTimeBalancer"

```
### Dynamic config
```yaml
http:
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

## K8s

### Chart config
```yaml
experimental:
  plugins:
    rtb_static:
        moduleName: "github.com/ArtemUgrimov/ResponseTimeBalancer"
        version: "v1.0.0"
```

### Middleware
```yaml
apiVersion: "traefik.io/v1alpha1"
kind: "Middleware"
metadata:
  name: "rtb"
spec:
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