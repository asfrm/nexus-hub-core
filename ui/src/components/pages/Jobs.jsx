import { Card, EmptyState } from '../ui';

export default function Jobs() {
  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-bold text-white">Job Queue</h2>
      <Card>
        <EmptyState
          icon="📋"
          title="Job Queue"
          description="Manage print jobs with thumbnails and real-time updates"
        />
      </Card>
    </div>
  );
}
