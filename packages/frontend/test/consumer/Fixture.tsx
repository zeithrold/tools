import type { FrontendPreferences, PreferencePolicy, LinkProps } from '@ztd-me/frontend'
import { useCallback, useState } from 'react'
import { FrontendProvider, PublicShell, ApplicationShell, useFrontendPreferences } from '@ztd-me/frontend/client'
import { ReviewContent, ReviewNavigation } from './ReviewContent.js'

export interface FixtureProps {
  initialPreferences: FrontendPreferences
  policy: PreferencePolicy
  application: boolean
  styleNonce: string
  footer: boolean
  review: boolean
}
function FixtureLink(props: LinkProps): React.JSX.Element {
  return <a {...props} data-router-link="fixture" />
}
function State({ persistenceErrors }: { persistenceErrors: number }): React.JSX.Element {
  const state = useFrontendPreferences()
  return (
    <>
      <h1>Shared shell fixture</h1>
      <output id="preferences">{JSON.stringify(state.preferences)}</output>
      <output id="resolved">{state.resolvedMode}</output>
      <output id="persistence">{state.persistence}</output>
      <output id="persistence-errors">{persistenceErrors}</output>
      <a href="/second">Second route</a>
      <button type="button" onClick={() => document.querySelector('#fullscreen-shell')?.requestFullscreen()}>Fullscreen</button>
      <textarea aria-label="Business draft" defaultValue="Keep this draft" />
    </>
  )
}
export function Fixture(props: FixtureProps): React.JSX.Element {
  const [persistenceErrors, setPersistenceErrors] = useState(0)
  const reportPersistenceError = useCallback(() => setPersistenceErrors(count => count + 1), [])
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(null)
  const common = {
    brand: { label: props.review ? 'Harbor Studio' : 'Fixture', homeHref: '/' },
    ...(props.review ? { projectActions: <ReviewNavigation /> } : {}),
    ...(props.footer ? {
      footer: {
        copyright: '© Harbor Studio',
        links: [
          { label: 'Source', href: 'https://codeberg.org/example/studio', ariaLabel: 'Studio source' },
          { label: 'support@harbor.example', href: 'mailto:support@harbor.example' },
        ],
      },
    } : {}),
    linkComponent: FixtureLink,
  }
  return (
    <div ref={setPortalContainer} id="fullscreen-shell" className={props.review ? 'review-document' : undefined}>
      <FrontendProvider
        initialPreferences={props.initialPreferences}
        policy={props.policy}
        portalContainer={portalContainer}
        styleNonce={props.styleNonce}
        onPersistenceError={reportPersistenceError}
      >
        {props.application ? (
          <ApplicationShell {...common}
            businessNavigation={<nav aria-label="Business"><a href="/workspace">Workspace</a></nav>}
            contextSidebar={<aside aria-label="Context">Project sidebar</aside>}
            serviceNotice={<p role="status">Synthetic service notice</p>}
          ><State persistenceErrors={persistenceErrors} /></ApplicationShell>
        ) : <PublicShell {...common}>
          {props.review ? <ReviewContent /> : <State persistenceErrors={persistenceErrors} />}
        </PublicShell>}
      </FrontendProvider>
    </div>
  )
}
