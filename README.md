# Fxiaoke OpenAPI SDK

Unofficial Go SDK for [the fxiaoke.com OpenAPI](https://open.fxiaoke.com/wiki.html)

## Install

```bash
go get -u github.com/k8scat/fxiaoke
```

## Quick start

```go
package main

import (
    "fmt"

    "github.com/k8scat/fxiaoke"
)

func main() {
    appID := ""
    appSecret := ""
    permanentCode := ""
    userID := ""
    corpID := ""
    client, err := fxiaoke.NewClient(appID, appSecret, permanentCode, userID, corpID)
    if err != nil {
        panic(err)
    }

    user, err := client.GetUserByOpenID(userID)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v", user)
}
```
