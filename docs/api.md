# API Design documentation

This document contains the API design. It is aimed at developers and to support administrators in debugging issues.

## API Responses

All API responses either have an emty body (for HTTP 204 and HTTP 404 responses) or the body consists of only JSON.

All API responses have **either** a `data` or an `error` top level key. They can’t appear at the same time.

The `error` key always has a value of type `string`, containing a human readable error message. Those error messages are intended to be passed to the user of the application.

The `data` key is either a list of objects (for collection endpoints) or a single object (for resource endpoints).

Unset attributes are not contained in the objects that the API returns. Unless an attribute is defined in here to be always contained in API responses, it is optional

### Reserved keys

The objects in the `data` key have several reserved keys that are read-only:

- `createdAt`: An RFC3339 timestamp of the time when the resource was created. Always present.
- `updatedAt`: An RFC3339 timestamp of the time when the resource was updated. Always present.
- `deletedAt`: An RFC3339 timestamp of the time when the resource was deleted.
- `id`: The ID of the object. IDs are unique for the backend instance. Always present.
