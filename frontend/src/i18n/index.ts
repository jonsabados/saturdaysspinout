import { createI18n } from 'vue-i18n'
import enUS from './locales/en-US.json'
import enGB from './locales/en-GB.json'
import esCO from './locales/es-CO.json'

export type SupportedLocale = 'en-US' | 'en-GB' | 'es-CO'

const STORAGE_KEY = 'user-locale'

function getInitialLocale(): SupportedLocale {
  // Check localStorage first (with guard for test environments)
  if (typeof localStorage !== 'undefined' && localStorage.getItem) {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'en-US' || stored === 'en-GB' || stored === 'es-CO') {
      return stored
    }
  }

  // Check browser preference
  const browserLang = typeof navigator !== 'undefined' ? navigator.language : 'en-US'
  if (browserLang === 'en-GB') {
    return 'en-GB'
  }
  if (browserLang.startsWith('es')) {
    return 'es-CO'
  }

  return 'en-US'
}

export const i18n = createI18n({
  legacy: false,
  locale: getInitialLocale(),
  fallbackLocale: 'en-US',
  messages: {
    'en-US': enUS,
    'en-GB': enGB,
    'es-CO': esCO,
  },
  datetimeFormats: {
    'en-US': {
      short: {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      },
      long: {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      },
    },
    'en-GB': {
      short: {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      },
      long: {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      },
    },
    'es-CO': {
      short: {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      },
      long: {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      },
    },
  },
  numberFormats: {
    'en-US': {
      decimal: {
        style: 'decimal',
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      },
    },
    'en-GB': {
      decimal: {
        style: 'decimal',
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      },
    },
    'es-CO': {
      decimal: {
        style: 'decimal',
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      },
    },
  },
})

export function setLocale(locale: SupportedLocale) {
  i18n.global.locale.value = locale
  localStorage.setItem(STORAGE_KEY, locale)
  document.documentElement.lang = locale
}

export function getLocale(): SupportedLocale {
  return i18n.global.locale.value as SupportedLocale
}