import { Card } from '../common/Card';

type SavingGoal = {
  id: string;
  title: string;
  progress: number;
  color: string;
};

type SavingGoalsCardProps = {
  goals: SavingGoal[];
  className?: string;
};

const size = 120;
const radius = 56;
const circumference = 2 * Math.PI * radius;

function renderProgress(progress: number) {
  return Math.min(Math.max(progress, 0), 100);
}

export function SavingGoalsCard({ goals, className }: SavingGoalsCardProps) {
  return (
    <Card className={className} title="Saving Goals">
      {goals.length > 0 ? (
        <div className="grid grid-cols-2 gap-8">
          {goals.map((goal) => {
            const progress = renderProgress(goal.progress);
            const dashOffset = circumference - (progress / 100) * circumference;

            return (
              <div key={goal.id} className="flex flex-col items-center gap-3">
                <div className="relative" style={{ width: size, height: size }}>
                  <svg className="absolute inset-0" width={size} height={size}>
                    <circle
                      cx="60"
                      cy="60"
                      r={radius}
                      fill="none"
                      stroke="rgba(148, 163, 184, 0.2)"
                      strokeWidth="8"
                    />
                  </svg>
                  <svg className="absolute inset-0 -rotate-90" width={size} height={size}>
                    <circle
                      cx="60"
                      cy="60"
                      r={radius}
                      fill="none"
                      stroke={goal.color}
                      strokeWidth="8"
                      strokeLinecap="round"
                      strokeDasharray={circumference}
                      strokeDashoffset={dashOffset}
                      className="transition-all duration-300"
                    />
                  </svg>
                  <div className="absolute inset-0 flex items-center justify-center text-2xl font-medium" style={{ color: goal.color }}>
                    {Math.round(progress)}%
                  </div>
                </div>
                <span className="text-sm font-medium text-text-primary">{goal.title}</span>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-sm text-text-muted">No saving goals yet.</p>
      )}
    </Card>
  );
}
