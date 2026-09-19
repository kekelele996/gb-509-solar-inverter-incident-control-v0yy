
import { EntityPage } from '../components/EntityPage';
import { ENTITY_CONFIGS } from '../types/status';
import { useFaultEventStore } from '../stores/fault-event';
export default function FaultEventPage() { return <EntityPage config={ENTITY_CONFIGS[2]} useStore={useFaultEventStore} />; }
