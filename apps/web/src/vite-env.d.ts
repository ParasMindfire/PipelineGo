/// <reference types="vite/client" />

// Typed env vars available via import.meta.env in the frontend
interface ImportMetaEnv {
  readonly VITE_API_URL: string
  readonly VITE_API_KEY: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
