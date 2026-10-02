import type { FrontendPreferences, PreferencePolicy, LinkProps } from '@ztd-me/frontend'
import { useState } from 'react'
import { FrontendProvider, PublicShell, ApplicationShell, useFrontendPreferences } from '@ztd-me/frontend/client'

export interface FixtureProps {
  initialPreferences: FrontendPreferences
  policy: PreferencePolicy
  application: boolean
  styleNonce: string
}
function FixtureLink(props: LinkProps): React.JSX.Element {
  return <a {...props} data-router-link="fixture" />
}
function State(): React.JSX.Element {
  const state = useFrontendPreferences()
  return (
    <>
      <h1>Shared shell fixture</h1>
      <output id="preferences">{JSON.stringify(state.preferences)}</output>
      <output id="resolved">{state.resolvedMode}</output>
      <output id="persistence">{state.persistence}</output>
      <a href="/second">Second route</a>
      <button type="button" onClick={() => document.querySelector('#fullscreen-shell')?.requestFullscreen()}>Fullscreen</button>
      <textarea aria-label="Business draft" defaultValue="Keep this draft" />
    </>
  )
}
export function Fixture(props: FixtureProps): React.JSX.Element {
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(null)
  const common = {
    brand: { label: 'Fixture', homeHref: '/' },
    repositoryUrl: 'https://github.com/zeithrold/website',
    linkComponent: FixtureLink,
  }
  return (
    <div ref={setPortalContainer} id="fullscreen-shell">
      <FrontendProvider
        initialPreferences={props.initialPreferences}
        policy={props.policy}
        portalContainer={portalContainer}
        styleNonce={props.styleNonce}
      >
        {props.application ? (
          <ApplicationShell {...common}
            businessNavigation={<nav aria-label="Business"><a href="/workspace">Workspace</a></nav>}
            contextSidebar={<aside aria-label="Context">Project sidebar</aside>}
            serviceNotice={<p role="status">Synthetic service notice</p>}
          ><State /></ApplicationShell>
        ) : <PublicShell {...common}><State /></PublicShell>}
      </FrontendProvider>
    </div>
  )
}
