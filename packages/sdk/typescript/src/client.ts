import { buildHttpError } from "./errors.js";
import { buildObjectPath, toAbsoluteUrl } from "./url.js";
import type { DOSClientOptions, PresignResponse, UploadInput } from "./types.js";

export class DOSClient {
  private readonly baseUrl: string;
  private readonly defaultHeaders: Record<string, string>;
  private readonly fetchImpl: typeof fetch;

  constructor(options: DOSClientOptions) {
    if (!options.baseUrl) {
      throw new Error("baseUrl is required");
    }

    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.defaultHeaders = options.defaultHeaders ?? {};
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  async uploadObject(input: UploadInput): Promise<void> {
    const path = buildObjectPath(this.baseUrl, "upload", input.bucket, input.objectKey);
    const headers: Record<string, string> = { ...this.defaultHeaders };

    if (input.contentType) {
      headers["Content-Type"] = input.contentType;
    }

    const response = await this.fetchImpl(path, {
      method: "POST",
      body: input.body,
      headers
    });

    if (!response.ok) {
      throw await buildHttpError(response);
    }
  }

  async createPresignedDownloadUrl(bucket: string, objectKey: string): Promise<string> {
    const path = buildObjectPath(this.baseUrl, "presign", bucket, objectKey);
    const response = await this.fetchImpl(path, {
      method: "GET",
      headers: this.defaultHeaders
    });

    if (!response.ok) {
      throw await buildHttpError(response);
    }

    const payload = (await response.json()) as PresignResponse;
    return toAbsoluteUrl(this.baseUrl, payload.url);
  }

  async downloadObject(bucket: string, objectKey: string): Promise<Response> {
    const url = await this.createPresignedDownloadUrl(bucket, objectKey);
    const response = await this.fetchImpl(url, {
      method: "GET",
      headers: this.defaultHeaders
    });

    if (!response.ok) {
      throw await buildHttpError(response);
    }

    return response;
  }
}
