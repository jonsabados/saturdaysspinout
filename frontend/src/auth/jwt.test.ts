import { describe, it, expect } from 'vitest'
import { decodeJWT, getEntitlementsFromToken } from './jwt'

describe('decodeJWT', () => {
  it('decodes a valid JWT payload', () => {
    // Create a test JWT with known payload
    // Header: {"alg":"HS256","typ":"JWT"}
    // Payload: {"sid":"test-session","uid":12345,"uname":"Test User","ent":["developer"],"iat":1700000000,"exp":1700086400}
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
    const payload = btoa(JSON.stringify({
      sid: 'test-session',
      uid: 12345,
      uname: 'Test User',
      ent: ['developer'],
      iat: 1700000000,
      exp: 1700086400,
    }))
    const signature = 'fake-signature'
    const token = `${header}.${payload}.${signature}`

    const result = decodeJWT(token)

    expect(result).toEqual({
      sid: 'test-session',
      uid: 12345,
      uname: 'Test User',
      ent: ['developer'],
      iat: 1700000000,
      exp: 1700086400,
    })
  })

  it('returns null for invalid token format', () => {
    expect(decodeJWT('not-a-jwt')).toBeNull()
    expect(decodeJWT('only.two')).toBeNull()
    expect(decodeJWT('')).toBeNull()
  })

  it('returns null for invalid base64', () => {
    expect(decodeJWT('valid.!!!invalid!!!.signature')).toBeNull()
  })
})

describe('getEntitlementsFromToken', () => {
  it('extracts entitlements from token', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
    const payload = btoa(JSON.stringify({
      sid: 'test',
      uid: 1,
      uname: 'Test',
      ent: ['developer', 'beta-tester'],
      iat: 1,
      exp: 2,
    }))
    const token = `${header}.${payload}.sig`

    expect(getEntitlementsFromToken(token)).toEqual(['developer', 'beta-tester'])
  })

  it('returns empty array when no entitlements', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
    const payload = btoa(JSON.stringify({
      sid: 'test',
      uid: 1,
      uname: 'Test',
      iat: 1,
      exp: 2,
    }))
    const token = `${header}.${payload}.sig`

    expect(getEntitlementsFromToken(token)).toEqual([])
  })

  it('returns empty array for invalid token', () => {
    expect(getEntitlementsFromToken('invalid')).toEqual([])
  })
})