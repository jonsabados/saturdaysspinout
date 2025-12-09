import { generatePKCE, storeCodeVerifier } from './pkce'

const IRACING_AUTHORIZE_URL = 'https://oauth.iracing.com/oauth2/authorize'

export async function initiateLogin(): Promise<void> {
  const clientId = import.meta.env.VITE_IRACING_CLIENT_ID
  if (!clientId) {
    throw new Error('VITE_IRACING_CLIENT_ID is not configured')
  }

  const { codeVerifier, codeChallenge } = await generatePKCE()
  storeCodeVerifier(codeVerifier)

  const redirectUri = `${window.location.origin}/auth/ir/callback`

  const params = new URLSearchParams({
    client_id: clientId,
    redirect_uri: redirectUri,
    response_type: 'code',
    scope: 'iracing.auth',
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  })

  window.location.href = `${IRACING_AUTHORIZE_URL}?${params.toString()}`
}