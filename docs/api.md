# API Design documentation

This document contains the API design. It is aimed at developers and to support administrators in debugging issues.

## High level guarantees

- Any resource will be available at the endpoints `/{resource}` for the collection and `/{resource}/{id}` for a single resource.
- Filtering on collection endpoints is implemented with URL paramaters (“query strings“)
- Collections always support the HTTP methods `GET` (read resources) and `POST` (create new resource)
- Resources support the HTTP methods `GET` (read resource), `PATCH` (update resource) and `DELETE` (delete resource)

## API responses

All API responses either have an empty body (only for HTTP 204 and some HTTP 404 responses) or the body consists of only JSON.

All API responses have **either** a `data` or an `error` top level key. They can’t appear at the same time.

The `error` key always has a value of type `string`, containing a human readable error message. Those error messages are intended to be passed to the user of the application.

The `data` key is either a list of objects (for collection endpoints) or a single object (for resource endpoints).

All attributes for a resource are always contained in the objects that the API returns.

## API resources

API resources share the following **read only** attributes in the `data` key:

- `createdAt` (string): An RFC3339 timestamp of the time when the resource was created.
- `updatedAt` (string): An RFC3339 timestamp of the time when the resource was updated.
- `deletedAt` (string): An RFC3339 timestamp of the time when the resource was deleted. Can be null.
- `id` (string): The UUID of the object.
- `links` (object): A map of related resources.
  - `self` (string): The full URL of the resource itself.
