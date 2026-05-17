export type PresignResponse = {
  url: string;
};

export type DOSClientOptions = {
  baseUrl: string;
  defaultHeaders?: Record<string, string>;
  fetchImpl?: typeof fetch;
};

export type UploadInput = {
  bucket: string;
  objectKey: string;
  body: BodyInit;
  contentType?: string;
};
