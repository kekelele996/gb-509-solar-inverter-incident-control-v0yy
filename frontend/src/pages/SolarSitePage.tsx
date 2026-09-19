
import { EntityPage } from '../components/EntityPage';
import { ENTITY_CONFIGS } from '../types/status';
import { useSolarSiteStore } from '../stores/solar-site';
export default function SolarSitePage() { return <EntityPage config={ENTITY_CONFIGS[0]} useStore={useSolarSiteStore} />; }
