export async function buildHttpError(response: Response): Promise<Error> {
  const body = await response.text();
  return new Error(`request failed (${response.status}): ${body || response.statusText}`);
}
