# TypeScript SDK

`@distributed-object-storage/sdk-typescript` provides a thin client for upload, presign, and download flows.

## Install

```bash
pnpm add @distributed-object-storage/sdk-typescript
```

## Usage

```ts
import { DOSClient } from "@distributed-object-storage/sdk-typescript";

const client = new DOSClient({
  baseUrl: "http://localhost:8080"
});

await client.uploadObject({
  bucket: "my-bucket",
  objectKey: "docs/hello.txt",
  body: "hello world",
  contentType: "text/plain"
});

const signedDownloadUrl = await client.createPresignedDownloadUrl(
  "my-bucket",
  "docs/hello.txt"
);
console.log(signedDownloadUrl);

const response = await client.downloadObject("my-bucket", "docs/hello.txt");
const text = await response.text();
console.log(text);
```

## Build

```bash
pnpm run build
```
