import WizardStep from "@/components/wizard/WizardStep";

export interface WelcomeFormValue {
  name: string;
  industry: string;
  company_size: string;
}

interface WelcomeStepProps {
  value: WelcomeFormValue;
  organizationName: string;
  onChange: (value: WelcomeFormValue) => void;
}

const industries = [
  "SaaS",
  "E-commerce",
  "FinTech",
  "Healthcare",
  "Education",
  "Professional Services",
  "Other",
];

const companySizes = ["1-10", "11-50", "51-200", "201-500", "500+"];

export default function WelcomeStep({
  value,
  organizationName,
  onChange,
}: WelcomeStepProps) {
  return (
    <WizardStep
      title="Welcome to Galdr"
      description="Let’s configure your organization so we can personalize scoring and insights."
    >
      <div className="galdr-panel mb-5 p-4 text-sm text-[var(--galdr-fg-muted)]">
        You’re about to connect customer data and generate your first health
        score preview. It usually takes just a few minutes.
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <label className="block">
          <span className="mb-1 block text-sm font-medium text-[var(--galdr-fg-muted)]">
            Organization name *
          </span>
          <input
            value={value.name}
            onChange={(e) =>
              onChange({
                ...value,
                name: e.target.value,
              })
            }
            placeholder={organizationName}
            className="galdr-input w-full px-3 py-2 text-sm"
          />
        </label>

        <label className="block">
          <span className="mb-1 block text-sm font-medium text-[var(--galdr-fg-muted)]">
            Industry
          </span>
          <select
            value={value.industry}
            onChange={(e) =>
              onChange({
                ...value,
                industry: e.target.value,
              })
            }
            className="galdr-input w-full px-3 py-2 text-sm"
          >
            <option value="">Select an industry</option>
            {industries.map((industry) => (
              <option key={industry} value={industry}>
                {industry}
              </option>
            ))}
          </select>
        </label>

        <label className="block md:col-span-2">
          <span className="mb-1 block text-sm font-medium text-[var(--galdr-fg-muted)]">
            Company size
          </span>
          <select
            value={value.company_size}
            onChange={(e) =>
              onChange({
                ...value,
                company_size: e.target.value,
              })
            }
            className="galdr-input w-full px-3 py-2 text-sm"
          >
            <option value="">Select company size</option>
            {companySizes.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
    </WizardStep>
  );
}
