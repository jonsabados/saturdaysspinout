import { describe, it, expect, beforeEach } from 'vitest'
import {
  generatePKCE,
  storeCodeVerifier,
  retrieveCodeVerifier,
  clearCodeVerifier,
} from './pkce'

describe('PKCE', () => {
  describe('generatePKCE', () => {
    it('generates a code verifier of 64 characters', async () => {
      const { codeVerifier } = await generatePKCE()
      expect(codeVerifier).toHaveLength(64)
    })

    it('generates a code verifier using only valid characters', async () => {
      const { codeVerifier } = await generatePKCE()
      const validChars = /^[A-Za-z0-9\-._~]+$/
      expect(codeVerifier).toMatch(validChars)
    })

    it('generates a base64url encoded code challenge', async () => {
      const { codeChallenge } = await generatePKCE()
      // base64url should not contain +, /, or =
      expect(codeChallenge).not.toMatch(/[+/=]/)
    })

    it('generates different values on each call', async () => {
      const first = await generatePKCE()
      const second = await generatePKCE()
      expect(first.codeVerifier).not.toBe(second.codeVerifier)
      expect(first.codeChallenge).not.toBe(second.codeChallenge)
    })

    it('generates a code challenge that is the SHA256 hash of the verifier', async () => {
      const { codeVerifier, codeChallenge } = await generatePKCE()

      // Manually compute the expected challenge
      const encoder = new TextEncoder()
      const data = encoder.encode(codeVerifier)
      const hash = await crypto.subtle.digest('SHA-256', data)
      const bytes = new Uint8Array(hash)
      let binary = ''
      for (const byte of bytes) {
        binary += String.fromCharCode(byte)
      }
      const expectedChallenge = btoa(binary)
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '')

      expect(codeChallenge).toBe(expectedChallenge)
    })
  })

  describe('sessionStorage helpers', () => {
    beforeEach(() => {
      sessionStorage.clear()
    })

    it('stores and retrieves a code verifier', () => {
      const verifier = 'test-verifier-12345'
      storeCodeVerifier(verifier)
      expect(retrieveCodeVerifier()).toBe(verifier)
    })

    it('returns null when no code verifier is stored', () => {
      expect(retrieveCodeVerifier()).toBeNull()
    })

    it('clears the stored code verifier', () => {
      storeCodeVerifier('test-verifier')
      clearCodeVerifier()
      expect(retrieveCodeVerifier()).toBeNull()
    })
  })
})