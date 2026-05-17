export function buildObjectPath(baseUrl: string, prefix: string, bucket: string, objectKey: string): string {
  const safeBucket = encodeURIComponent(bucket);
  const safeKey = objectKey
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");

  return `${baseUrl}/${prefix}/${safeBucket}/${safeKey}`;
}

export function toAbsoluteUrl(baseUrl: string, url: string): string {
  if (url.startsWith("http://") || url.startsWith("https://")) {
    return url;
  }

  return `${baseUrl}${url.startsWith("/") ? "" : "/"}${url}`;
}
