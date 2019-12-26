go-circuitbreaker
===
[![GoDoc](https://godoc.org/github.com/nasa9084/go-circuitbreaker?status.svg)](https://godoc.org/github.com/nasa9084/go-circuitbreaker)

An implementation of Circuit Breaker pattern for Go

## SYNOPSIS

``` go
package main

import "github.com/nasa9084/go-circuitbreaker"

func main() {
    cb := circuitbreaker.New()

    for {
        if cb.IsAvail() {
            if err := Func(); err != nil {
                cb.Fail()
                log.Print(err)
                continue
            }
            cb.Success()
        }
    }
}
```

## DESCRIPTION

This package is an implementation of Circuit Breaker pattern in idiomatic Go way. This circuit breaker can be used as conditional variable and you can retry without considering about the circuit state.
