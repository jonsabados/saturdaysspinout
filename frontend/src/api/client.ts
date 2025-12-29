import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export interface ApiError {
  message: string
  correlationId?: string
}

export interface Race {
  id: number
  subsessionId: number
  trackId: number
  seriesId: number
  seriesName: string
  carId: number
  startTime: string
  startPosition: number
  startPositionInClass: number
  finishPosition: number
  finishPositionInClass: number
  incidents: number
  oldCpi: number
  newCpi: number
  oldIrating: number
  newIrating: number
  oldLicenseLevel: number
  newLicenseLevel: number
  oldSubLevel: number
  newSubLevel: number
  reasonOut: string
}

export interface Pagination {
  page: number
  resultsPerPage: number
  totalResults: number
  totalPages: number
}

export interface RacesResponse {
  items: Race[]
  pagination: Pagination
  correlationId: string
}

export interface RaceResponse {
  response: Race
  correlationId: string
}

export interface Driver {
  driverId: number
  driverName: string
  memberSince: string
  racesIngestedTo: string | null
  ingestionBlockedUntil: string | null
  firstLogin: string
  lastLogin: string
  loginCount: number
  sessionCount: number
}

export interface DriverResponse {
  response: Driver
  correlationId: string
}

export interface TrackMapLayers {
  background: string
  inactive: string
  active: string
  pitroad: string
  startFinish: string
  turns: string
}

export interface Track {
  id: number
  name: string
  configName: string
  category: string
  location: string
  cornersPerLap: number
  lengthMiles: number
  description: string
  logoUrl: string
  smallImageUrl: string
  largeImageUrl: string
  trackMapUrl: string
  trackMapLayers: TrackMapLayers
  isDirt: boolean
  isOval: boolean
  hasNightLighting: boolean
  rainEnabled: boolean
  freeWithSubscription: boolean
  retired: boolean
  pitRoadSpeedLimit: number
}

export interface TracksResponse {
  response: Track[]
  correlationId: string
}

export interface Car {
  id: number
  name: string
  nameAbbreviated: string
  make: string
  model: string
  description: string
  weight: number
  hpUnderHood: number
  hpActual: number
  categories: string[]
  logoUrl: string
  smallImageUrl: string
  largeImageUrl: string
  hasHeadlights: boolean
  hasMultipleDryTires: boolean
  rainEnabled: boolean
  freeWithSubscription: boolean
  retired: boolean
}

export interface CarsResponse {
  response: Car[]
  correlationId: string
}

// Session/Race Details types
export interface SessionAllowedLicense {
  groupName: string
  licenseGroup: number
  maxLicenseLevel: number
  minLicenseLevel: number
  parentId: number
}

export interface SessionCarInClass {
  carId: number
}

export interface SessionCarClass {
  carClassId: number
  shortName: string
  name: string
  strengthOfField: number
  numEntries: number
  carsInClass: SessionCarInClass[]
}

export interface SessionRaceSummary {
  subsessionId: number
  averageLap: number
  lapsComplete: number
  numCautions: number
  numCautionLaps: number
  numLeadChanges: number
  fieldStrength: number
  numOptLaps: number
  hasOptPath: boolean
  specialEventType: number
  specialEventTypeText: string
}

export interface SessionHelmet {
  pattern: number
  color1: string
  color2: string
  color3: string
  faceType: number
  helmetType: number
}

export interface SessionLivery {
  carId: number
  pattern: number
  color1: string
  color2: string
  color3: string
  numberFont: number
  numberColor1: string
  numberColor2: string
  numberColor3: string
  numberSlant: number
  sponsor1: number
  sponsor2: number
  carNumber: string
  wheelColor: string | null
  rimType: number
}

export interface SessionSuit {
  pattern: number
  color1: string
  color2: string
  color3: string
}

export interface SessionDriverResult {
  custId: number
  displayName: string
  aggregateChampPoints: number
  ai: boolean
  averageLap: number
  bestLapNum: number
  bestLapTime: number
  bestNlapsNum: number
  bestNlapsTime: number
  bestQualLapAt: string
  bestQualLapNum: number
  bestQualLapTime: number
  carClassId: number
  carClassName: string
  carClassShortName: string
  carId: number
  carName: string
  carCfg: number
  champPoints: number
  classInterval: number
  countryCode: string
  division: number
  divisionName: string
  dropRace: boolean
  finishPosition: number
  finishPositionInClass: number
  flairId: number
  flairName: string
  flairShortname: string
  friend: boolean
  helmet: SessionHelmet
  incidents: number
  interval: number
  lapsComplete: number
  lapsLead: number
  leagueAggPoints: number
  leaguePoints: number
  licenseChangeOval: number
  licenseChangeRoad: number
  livery: SessionLivery
  maxPctFuelFill: number
  newCpi: number
  newLicenseLevel: number
  newSubLevel: number
  newTtrating: number
  newIrating: number
  oldCpi: number
  oldLicenseLevel: number
  oldSubLevel: number
  oldTtrating: number
  oldIrating: number
  optLapsComplete: number
  position: number
  qualLapTime: number
  reasonOut: string
  reasonOutId: number
  startingPosition: number
  startingPositionInClass: number
  suit: SessionSuit
  watched: boolean
  weightPenaltyKg: number
}

export interface SessionWeatherResult {
  avgSkies: number
  avgCloudCoverPct: number
  minCloudCoverPct: number
  maxCloudCoverPct: number
  tempUnits: number
  avgTemp: number
  minTemp: number
  maxTemp: number
  avgRelHumidity: number
  windUnits: number
  avgWindSpeed: number
  minWindSpeed: number
  maxWindSpeed: number
  avgWindDir: number
  maxFog: number
  fogTimePct: number
  precipTimePct: number
  precipMm: number
  precipMm2hrBeforeSession: number
  simulatedStartTime: string
}

export interface SessionSimResult {
  simsessionNumber: number
  simsessionName: string
  simsessionType: number
  simsessionTypeName: string
  simsessionSubtype: number
  weatherResult: SessionWeatherResult
  results: SessionDriverResult[]
}

export interface SessionSplit {
  subsessionId: number
  eventStrengthOfField: number
}

export interface SessionTrack {
  trackId: number
  trackName: string
  configName: string
  category: string
  categoryId: number
}

export interface SessionTrackState {
  leaveMarbles: boolean
  practiceRubber: number
  qualifyRubber: number
  raceRubber: number
  warmupRubber: number
}

export interface SessionWeather {
  allowFog: boolean
  fog: number
  precipMm2hrBeforeFinalSession: number
  precipMmFinalSession: number
  precipOption: number
  precipTimePct: number
  relHumidity: number
  simulatedStartTime: string
  skies: number
  tempUnits: number
  tempValue: number
  timeOfDay: number
  trackWater: number
  type: number
  version: number
  weatherVarInitial: number
  weatherVarOngoing: number
  windDir: number
  windUnits: number
  windValue: number
}

export interface Session {
  subsessionId: number
  sessionId: number
  allowedLicenses: SessionAllowedLicense[]
  associatedSubsessionIds: number[]
  canProtest: boolean
  carClasses: SessionCarClass[]
  cautionType: number
  cooldownMinutes: number
  cornersPerLap: number
  damageModel: number
  driverChangeParam1: number
  driverChangeParam2: number
  driverChangeRule: number
  driverChanges: boolean
  endTime: string
  eventAverageLap: number
  eventBestLapTime: number
  eventLapsComplete: number
  eventStrengthOfField: number
  eventType: number
  eventTypeName: string
  heatInfoId: number
  licenseCategory: string
  licenseCategoryId: number
  limitMinutes: number
  maxTeamDrivers: number
  maxWeeks: number
  minTeamDrivers: number
  numCautionLaps: number
  numCautions: number
  numDrivers: number
  numLapsForQualAverage: number
  numLapsForSoloAverage: number
  numLeadChanges: number
  officialSession: boolean
  pointsType: string
  privateSessionId: number
  raceSummary: SessionRaceSummary
  raceWeekNum: number
  resultsRestricted: boolean
  seasonId: number
  seasonName: string
  seasonQuarter: number
  seasonShortName: string
  seasonYear: number
  seriesId: number
  seriesLogo: string
  seriesName: string
  seriesShortName: string
  sessionResults: SessionSimResult[]
  sessionSplits: SessionSplit[]
  specialEventType: number
  startTime: string
  track: SessionTrack
  trackState: SessionTrackState
  weather: SessionWeather
}

export interface SessionResponse {
  response: Session
  correlationId: string
}

export class ApiClient {
  private authStore: ReturnType<typeof useAuthStore>
  private sessionStore: ReturnType<typeof useSessionStore>

  constructor(authStore: ReturnType<typeof useAuthStore>, sessionStore: ReturnType<typeof useSessionStore>) {
    this.authStore = authStore
    this.sessionStore = sessionStore
  }

  async fetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    if (!this.authStore.isLoggedIn) {
      throw new Error('Not authenticated')
    }

    const headers = new Headers(options.headers)
    headers.set('Authorization', `Bearer ${this.authStore.token}`)

    const response = await fetch(`${apiBaseUrl}${path}`, {
      ...options,
      headers,
    })

    if (response.status === 401) {
      // Token might have expired between check and request - try refresh once
      const refreshed = await this.authStore.refreshToken()
      if (refreshed) {
        // Retry the request with new token
        headers.set('Authorization', `Bearer ${this.authStore.token}`)
        const retryResponse = await fetch(`${apiBaseUrl}${path}`, {
          ...options,
          headers,
        })
        if (!retryResponse.ok) {
          throw await this.parseError(retryResponse)
        }
        return retryResponse.json()
      }
      throw new Error('Session expired - please log in again')
    }

    if (!response.ok) {
      throw await this.parseError(response)
    }

    return response.json()
  }

  private async parseError(response: Response): Promise<Error> {
    try {
      const data = await response.json() as ApiError
      let message = data.message || `Request failed: ${response.status}`
      if (data.correlationId) {
        message += ` (Correlation ID: ${data.correlationId})`
      }
      return new Error(message)
    } catch {
      return new Error(`Request failed: ${response.status}`)
    }
  }

  async triggerRaceIngestion(): Promise<{ throttled: boolean; retryAfter?: number }> {
    if (!this.sessionStore.isReady) {
      throw new Error('Session not ready')
    }

    if (!this.authStore.isLoggedIn) {
      throw new Error('Not authenticated')
    }

    const headers = new Headers()
    headers.set('Authorization', `Bearer ${this.authStore.token}`)
    headers.set('Content-Type', 'application/json')

    const response = await fetch(`${apiBaseUrl}/ingestion/race`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ notifyConnectionId: this.sessionStore.connectionId }),
    })

    if (response.status === 429) {
      const data = await response.json()
      const retryAfter = data.retryAfter ?? parseInt(response.headers.get('Retry-After') || '0', 10)
      console.warn(`[ApiClient] Race ingestion throttled - already in progress. Retry after ${retryAfter} seconds.`)
      return { throttled: true, retryAfter }
    }

    if (!response.ok) {
      throw await this.parseError(response)
    }

    return { throttled: false }
  }

  async getRaces(
    driverId: number,
    startTime: Date,
    endTime: Date,
    page = 1,
    resultsPerPage = 10
  ): Promise<RacesResponse> {
    const params = new URLSearchParams({
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
      page: page.toString(),
      resultsPerPage: resultsPerPage.toString(),
    })
    return this.fetch<RacesResponse>(`/driver/${driverId}/races?${params}`)
  }

  async getRace(driverId: number, driverRaceId: number): Promise<RaceResponse> {
    return this.fetch<RaceResponse>(`/driver/${driverId}/races/${driverRaceId}`)
  }

  async getDriver(driverId: number): Promise<DriverResponse> {
    return this.fetch<DriverResponse>(`/driver/${driverId}`)
  }

  async getTracks(): Promise<TracksResponse> {
    return this.fetch<TracksResponse>('/tracks')
  }

  async getCars(): Promise<CarsResponse> {
    return this.fetch<CarsResponse>('/cars')
  }

  async getSession(subsessionId: number): Promise<SessionResponse> {
    return this.fetch<SessionResponse>(`/session/${subsessionId}`)
  }

  async deleteDriverRaces(driverId: number): Promise<void> {
    if (!this.authStore.isLoggedIn) {
      throw new Error('Not authenticated')
    }

    const headers = new Headers()
    headers.set('Authorization', `Bearer ${this.authStore.token}`)

    const response = await fetch(`${apiBaseUrl}/driver/${driverId}/races`, {
      method: 'DELETE',
      headers,
    })

    if (response.status === 401) {
      const refreshed = await this.authStore.refreshToken()
      if (refreshed) {
        headers.set('Authorization', `Bearer ${this.authStore.token}`)
        const retryResponse = await fetch(`${apiBaseUrl}/driver/${driverId}/races`, {
          method: 'DELETE',
          headers,
        })
        if (!retryResponse.ok) {
          throw await this.parseError(retryResponse)
        }
        return
      }
      throw new Error('Session expired - please log in again')
    }

    if (!response.ok) {
      throw await this.parseError(response)
    }
  }
}

export function useApiClient() {
  const authStore = useAuthStore()
  const sessionStore = useSessionStore()
  return new ApiClient(authStore, sessionStore)
}