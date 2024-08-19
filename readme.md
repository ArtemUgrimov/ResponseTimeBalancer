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
          cookiePodIdName: "pod-id"
          additionalCookie: "Path=/"
          responseTimeHeaderName: "Tm"
          responseTimeDiffMsToInvalidate: "30"
          logStart: true
          logCookieFound: false
          logCookieResetToBest: true
          logIdle: false
          logBestUpdate: true
```

## K8s

### Chart config
```yaml
experimental:
  plugins:
    rtb_static:
        moduleName: "github.com/ArtemUgrimov/ResponseTimeBalancer"
        version: "v1.0.9"
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
      cookiePodIdName: "pod-id"
      additionalCookie: "; Path=/"
      responseTimeHeaderName: "Tm"
      responseTimeDiffMsToInvalidate: "30"
      logStart: true
      logCookieFound: false
      logCookieResetToBest: true
      logIdle: false
      logBestUpdate: true
```