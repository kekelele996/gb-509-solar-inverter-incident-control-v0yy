
import { EntityPage } from '../components/EntityPage';
import { ENTITY_CONFIGS } from '../types/status';
import { useInverterUnitStore } from '../stores/inverter-unit';
export default function InverterUnitPage() { return <EntityPage config={ENTITY_CONFIGS[1]} useStore={useInverterUnitStore} />; }
