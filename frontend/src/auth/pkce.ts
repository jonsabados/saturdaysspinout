const CODE_VERIFIER_KEY = 'iracing_oauth_code_verifier'

function generateRandomString(length: number): string {
  const array = new Uint8Array(length)
  crypto.getRandomValues(array)
  // Use URL-safe base64 characters
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~'
  return Array.from(array, (byte) => chars[byte % chars.length]).join('')
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  const data = encoder.encode(plain)
  return crypto.subtle.digest('SHA-256', data)
}

function base64UrlEncode(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

export async function generatePKCE(): Promise<{ codeVerifier: string; codeChallenge: string }> {
  const codeVerifier = generateRandomString(64)
  const hash = await sha256(codeVerifier)
  const codeChallenge = base64UrlEncode(hash)
  return { codeVerifier, codeChallenge }
}

export function storeCodeVerifier(codeVerifier: string): void {
  sessionStorage.setItem(CODE_VERIFIER_KEY, codeVerifier)
}

export function retrieveCodeVerifier(): string | null {
  return sessionStorage.getItem(CODE_VERIFIER_KEY)
}

export function clearCodeVerifier(): void {
  sessionStorage.removeItem(CODE_VERIFIER_KEY)
}