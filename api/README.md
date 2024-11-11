# Rudder API SDK - Go

A Golang SDK implementation for the Rudder Public API.

## Overview

This library can be used to interface with Rudder Public API v2. Initially, it will support CRUD operations
for common Rudder resources (e.g Sources, Destinations, Connections). Eventually, additional support will be added
for any features available by Rudder Public API.

  **This library, as well as Rudder Public API v2, is still in experimental alpha stage. Although we design for long-term backwards compatibility support in mind, some breaking changes are expected at this point.**

## Features

* Supports CRUD operations for Sources, Destinations and Connections

## Getting started

```Golang
// create a client
c, err := client.New("my-access-token")

// check for any errors
if err != nil {
  return err
}

// fetch a Source by ID
src, err := c.Sources.Get(context.Background(), "some-id")

// list all Sources
page, err := c.Sources.List(context.Background())
if err == nil {
  return err  
}
for page != nil {
  fmt.Println(page.Sources)
  page, err = c.Sources.Next(context.Background(), page.Paging)
  if err != nil {
    return err
  }
}

// create a new Destination
dst, err := c.Destinations.Create(context.Background(), &Destination{
  Type: 'POSTGRES',
  Name: 'my postgres',
  Config: json.RawMessage(`{
    "host": "example.com",
    "username": "rudder",
    "password": "some secret",
    "port": "5432"
  }`)
})

// check the error
if apierr, ok := err.(*client.APIError); ok {
  fmt.Println("status code:", apierr.HTTPStatusCode)
  fmt.Println("error message:", apierr.Message)
  return apierr
}

// create a new Connection
conn, err := c.Connections.Create(context.Background(), &Connection{
  SourceID: src.ID,
  DestinationID: dst.ID,
})
```

## License

The RudderStack API Go SDK is released under the [**MIT License**](https://opensource.org/licenses/MIT).
