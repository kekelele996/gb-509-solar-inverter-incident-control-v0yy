
import { EntityPage } from '../components/EntityPage';
import { ENTITY_CONFIGS } from '../types/status';
import { useMitigationActionStore } from '../stores/mitigation-action';
export default function MitigationActionPage() { return <EntityPage config={ENTITY_CONFIGS[3]} useStore={useMitigationActionStore} />; }
