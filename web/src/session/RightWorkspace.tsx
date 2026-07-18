import { useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { patchSettings, reanalyzeSession } from '../api'
import type { GameContext } from '../types'
import { AdventuresPanel } from '../AdventuresPanel'
import { CalendarPanel } from '../CalendarPanel'
import { CompendiumPanel } from '../CompendiumPanel'
import { DecksPanel } from '../DecksPanel'
import { FactionsPanel } from '../FactionsPanel'
import { GMToolsPanel } from '../GMToolsPanel'
import { HandoutsPanel } from '../HandoutsPanel'
import { JournalPanel } from '../JournalPanel'
import { NPCRosterPanel } from '../NPCRosterPanel'
import { NPCStatBlockPanel } from '../NPCStatBlockPanel'
import { ObjectivesPanel } from '../ObjectivesPanel'
import { OraclePanel } from '../OraclePanel'
import { RelationshipsPanel } from '../RelationshipsPanel'
import { SecretsPanel } from '../SecretsPanel'
import { SessionTimeline } from '../SessionTimeline'
import { WorldNotesPanel } from '../WorldNotesPanel'
import { WorkspaceNavigation } from '../navigation/WorkspaceNavigation'
import type { MobileDestination, PanelID } from '../navigation/panelRegistry'
import { useToast } from '../ui/ToastProvider'

export interface RightWorkspaceProps {
  aiEnabled: boolean
  ctx: GameContext
  lastEvent: unknown
  mobileDestination: MobileDestination
  onMobileDestinationChange: (destination: MobileDestination) => void
  rightTab: PanelID
  setRightTab: Dispatch<SetStateAction<PanelID>>
}

export function RightWorkspace({
  aiEnabled,
  ctx,
  lastEvent,
  mobileDestination,
  onMobileDestinationChange,
  rightTab,
  setRightTab,
}: RightWorkspaceProps) {
  const toast = useToast()
  const [journalSubTab, setJournalSubTab] = useState<'notes' | 'timeline'>('notes')

  return (
    <aside className="sidebar-right">
      <WorkspaceNavigation
        activePanel={rightTab}
        onPanelChange={setRightTab}
        mobileDestination={mobileDestination}
        onMobileDestinationChange={onMobileDestinationChange}
        mobile={false}
      />
      <div
        className="tab-content"
        id="workspace-active-panel"
        role="region"
        aria-labelledby={`workspace-panel-control-${rightTab}`}
      >
        {rightTab === 'handouts' && ctx.campaign && (
          <HandoutsPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} />
        )}
        {rightTab === 'compendium' && ctx.campaign && (
          <CompendiumPanel rulesetId={ctx.campaign.ruleset_id} />
        )}
        {rightTab === 'decks' && ctx.campaign && ctx.session && (
          <DecksPanel campaignId={ctx.campaign.id} sessionId={ctx.session.id} lastEvent={lastEvent} />
        )}
        {rightTab === 'notes' && ctx.campaign && (
          <WorldNotesPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} aiEnabled={aiEnabled} />
        )}
        {rightTab === 'journal' && (
          <div className="journal-container">
            <div className="journal-subtabs">
              <button className={`journal-subtab${journalSubTab === 'notes' ? ' active' : ''}`} onClick={() => setJournalSubTab('notes')}>Notes</button>
              <button className={`journal-subtab${journalSubTab === 'timeline' ? ' active' : ''}`} onClick={() => setJournalSubTab('timeline')}>Timeline</button>
            </div>
            <button
              className="journal-reanalyze-btn"
              onClick={async () => {
                if (!ctx.session?.id) return
                try {
                  await reanalyzeSession(ctx.session.id)
                  toast.success('Session reanalysis started.')
                } catch (cause) {
                  console.error(cause)
                  toast.error('The session could not be reanalyzed. Try again.')
                }
              }}
              title="Re-analyze session for objectives and NPCs"
            >
              ↻ Reanalyze
            </button>
            {journalSubTab === 'notes' ? (
              <JournalPanel
                session={ctx.session ?? null}
                campaignId={ctx.campaign?.id ?? null}
                lastEvent={lastEvent}
                aiEnabled={aiEnabled}
              />
            ) : ctx.session?.id != null ? (
              <SessionTimeline sessionId={ctx.session.id} lastEvent={lastEvent} />
            ) : null}
          </div>
        )}
        {rightTab === 'npcs' && <NPCRosterPanel sessionId={ctx.session?.id ?? null} lastEvent={lastEvent} />}
        {rightTab === 'objectives' && (
          <ObjectivesPanel campaignId={ctx.campaign?.id ?? null} sessionId={ctx.session?.id ?? null} lastEvent={lastEvent} />
        )}
        {rightTab === 'oracle' && ctx.session && <OraclePanel sessionId={ctx.session.id} />}
        {rightTab === 'relationships' && ctx.campaign && (
          <RelationshipsPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} />
        )}
        {rightTab === 'factions' && ctx.campaign && <FactionsPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} />}
        {rightTab === 'calendar' && ctx.campaign && (
          <CalendarPanel campaignId={ctx.campaign.id} sessionId={ctx.session?.id ?? null} lastEvent={lastEvent} />
        )}
        {rightTab === 'npcstats' && ctx.campaign && (
          <NPCStatBlockPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} />
        )}
        {rightTab === 'adventures' && ctx.campaign && (
          <AdventuresPanel
            campaignId={ctx.campaign.id}
            onSessionClick={async (sessionId: number) => {
              try {
                await patchSettings({ session_id: sessionId })
              } catch (cause) {
                console.error(cause)
                toast.error('Could not open that adventure session.')
              }
            }}
            lastEvent={lastEvent}
          />
        )}
        {rightTab === 'secrets' && ctx.campaign && (
          <SecretsPanel campaignId={ctx.campaign.id} sessionId={ctx.session?.id ?? null} lastEvent={lastEvent} />
        )}
        {rightTab === 'gmtools' && (
          <GMToolsPanel sessionId={ctx.session?.id ?? null} campaignId={ctx.campaign?.id ?? null} aiEnabled={aiEnabled} />
        )}
      </div>
    </aside>
  )
}
