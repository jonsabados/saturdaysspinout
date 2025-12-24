/**
 * JWT payload structure matching backend SessionClaims
 */
export interface JWTPayload {
  sid: string          // session ID
  uid: number          // iRacing user ID
  uname: string        // iRacing username
  ent?: string[]       // entitlements (optional)
  iat: number          // issued at
  exp: number          // expiration
}

/**
 * Decodes a JWT token and returns the payload.
 * Note: This does NOT verify the signature - that's the backend's job.
 * We're just reading the claims that were already verified when the token was issued.
 */
export function decodeJWT(token: string): JWTPayload | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) {
      return null
    }

    // JWT uses base64url encoding (- instead of +, _ instead of /)
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = atob(base64)
    return JSON.parse(jsonPayload)
  } catch {
    return null
  }
}

/**
 * Extracts entitlements from a JWT token.
 * Returns an empty array if the token is invalid or has no entitlements.
 */
export function getEntitlementsFromToken(token: string): string[] {
  const payload = decodeJWT(token)
  return payload?.ent ?? []
}