import { useMemo, useState } from 'react'
import type { SDKCredentialKind } from '../../sdkconfig/types'

type QuickstartID = 'python' | 'node' | 'go' | 'dotnet' | 'browser' | 'react'

interface SDKQuickstartProps {
  kind: SDKCredentialKind
  sdkKey: string
}

interface Quickstart {
  id: QuickstartID
  label: string
  install: string
  code: string
}

export function SDKQuickstart({ kind, sdkKey }: SDKQuickstartProps) {
  const baseUrl = typeof window === 'undefined' ? 'https://flags.example.com' : window.location.origin
  const quickstarts = useMemo<Quickstart[]>(() => (
    kind === 'server'
      ? serverQuickstarts(baseUrl)
      : clientQuickstarts(baseUrl, sdkKey)
  ), [baseUrl, kind, sdkKey])
  const [selectedID, setSelectedID] = useState<QuickstartID>(quickstarts[0]?.id ?? 'node')
  const selected = quickstarts.find((quickstart) => quickstart.id === selectedID) ?? quickstarts[0]
  const [copied, setCopied] = useState<'key' | 'install' | 'code'>()

  if (!selected) {
    return null
  }

  async function copy(value: string, target: 'key' | 'install' | 'code') {
    await navigator.clipboard.writeText(value)
    setCopied(target)
    window.setTimeout(() => setCopied((current) => current === target ? undefined : current), 1500)
  }

  return (
    <div className="mt-4 rounded-xl border border-slate-800 bg-slate-950/70">
      <div className="flex flex-col gap-3 border-b border-slate-800 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <strong className="text-xs font-semibold text-slate-200">Use this key</strong>
          <p className="mt-1 text-[11px] leading-5 text-slate-500">
            {kind === 'server'
              ? 'Keep the server key out of source control. Put it in your application secret/environment configuration.'
              : 'Client keys are public identifiers. Only flags marked client-visible are delivered to them.'}
          </p>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {quickstarts.map((quickstart) => (
            <button
              className={`rounded-md border px-2.5 py-1 text-[11px] font-semibold transition ${
                selected.id === quickstart.id
                  ? 'border-switchonyourcode-500/60 bg-switchonyourcode-500/10 text-switchonyourcode-300'
                  : 'border-slate-800 text-slate-500 hover:border-slate-700 hover:text-slate-300'
              }`}
              key={quickstart.id}
              onClick={() => setSelectedID(quickstart.id)}
              type="button"
            >
              {quickstart.label}
            </button>
          ))}
        </div>
      </div>

      {kind === 'server' ? (
        <CodeBlock
          action={copied === 'key' ? 'Copied' : 'Copy'}
          label="Environment variable"
          onCopy={() => void copy(`SWITCHONYOURCODE_SDK_KEY=${sdkKey}`, 'key')}
          value={`SWITCHONYOURCODE_SDK_KEY=${sdkKey}`}
        />
      ) : null}
      <CodeBlock
        action={copied === 'install' ? 'Copied' : 'Copy'}
        label="Install"
        onCopy={() => void copy(selected.install, 'install')}
        value={selected.install}
      />
      <CodeBlock
        action={copied === 'code' ? 'Copied' : 'Copy'}
        label="Configure and evaluate"
        onCopy={() => void copy(selected.code, 'code')}
        value={selected.code}
      />

      <div className="border-t border-slate-800 px-4 py-3 text-[11px] leading-5 text-slate-500">
        Flag evaluation happens locally from cached schema-v1 configuration. Keep a caller fallback for startup/failure safety; long-running SDKs can use conditional refreshes and the realtime invalidation transport without adding a network request to each evaluation.
      </div>
    </div>
  )
}

function CodeBlock({ action, label, onCopy, value }: { action: string; label: string; onCopy: () => void; value: string }) {
  return (
    <div className="border-b border-slate-800 px-4 py-3 last:border-b-0">
      <div className="mb-2 flex items-center justify-between gap-3">
        <span className="text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-600">{label}</span>
        <button className="text-[11px] font-medium text-slate-500 transition hover:text-slate-300" onClick={onCopy} type="button">
          {action}
        </button>
      </div>
      <pre className="overflow-x-auto whitespace-pre rounded-lg bg-slate-950 p-3 text-[11px] leading-5 text-slate-300"><code>{value}</code></pre>
    </div>
  )
}

function serverQuickstarts(baseUrl: string): Quickstart[] {
  return [
    {
      id: 'python',
      label: 'Python',
      install: 'pip install switchonyourcode',
      code: `import os\nfrom switchonyourcode import SwitchOnYourCodeClient\n\nflags = SwitchOnYourCodeClient(\n    base_url=${JSON.stringify(baseUrl)},\n    server_key=os.environ["SWITCHONYOURCODE_SDK_KEY"],\n)\nflags.initialize()\n\nenabled = flags.get_boolean_value(\n    "your-flag-key",\n    False,\n    {"targetingKey": "user-123"},\n)`,
    },
    {
      id: 'node',
      label: 'Node.js',
      install: 'pnpm add @switchonyourcode/node',
      code: `import { createNodeClient } from '@switchonyourcode/node'\n\nconst flags = await createNodeClient({\n  baseUrl: ${JSON.stringify(baseUrl)},\n  serverKey: process.env.SWITCHONYOURCODE_SDK_KEY!,\n  autoPoll: true,\n})\n\nconst enabled = flags.getBooleanValue('your-flag-key', false, {\n  targetingKey: 'user-123',\n})`,
    },
    {
      id: 'go',
      label: 'Go',
      install: 'go get github.com/switchonyourcode/sdk-go',
      code: `ctx := context.Background()\nflags, err := switchonyourcode.NewClientAndWait(ctx, switchonyourcode.ClientOptions{\n    BaseURL: ${JSON.stringify(baseUrl)},\n    ServerKey: os.Getenv("SWITCHONYOURCODE_SDK_KEY"),\n})\nif err != nil {\n    log.Fatal(err)\n}\ndefer flags.Close()\n\nenabled := flags.Boolean("your-flag-key", false, switchonyourcode.EvaluationContext{\n    TargetingKey: "user-123",\n})`,
    },
    {
      id: 'dotnet',
      label: '.NET',
      install: 'dotnet add package SwitchOnYourCode',
      code: `using SwitchOnYourCode;\n\nawait using var flags = await SwitchOnYourCodeClient.CreateAndWaitAsync(\n    new SwitchOnYourCodeClientOptions\n    {\n        BaseUrl = ${JSON.stringify(baseUrl)},\n        ServerKey = Environment.GetEnvironmentVariable("SWITCHONYOURCODE_SDK_KEY")!,\n    });\n\nvar enabled = flags.GetBooleanValue(\n    "your-flag-key",\n    fallback: false,\n    new EvaluationContext(TargetingKey: "user-123"));`,
    },
  ]
}

function clientQuickstarts(baseUrl: string, sdkKey: string): Quickstart[] {
  return [
    {
      id: 'browser',
      label: 'Browser',
      install: 'pnpm add @switchonyourcode/browser',
      code: `import { createBrowserClient } from '@switchonyourcode/browser'\n\nconst flags = await createBrowserClient({\n  baseUrl: ${JSON.stringify(baseUrl)},\n  clientKey: ${JSON.stringify(sdkKey)},\n})\n\nconst enabled = flags.getBooleanValue('your-flag-key', false, {\n  targetingKey: 'user-123',\n})`,
    },
    {
      id: 'react',
      label: 'React',
      install: 'pnpm add @switchonyourcode/browser @switchonyourcode/react',
      code: `import { createBrowserClient } from '@switchonyourcode/browser'\nimport { SwitchOnYourCodeProvider, useBooleanFlag } from '@switchonyourcode/react'\n\nconst flags = await createBrowserClient({\n  baseUrl: ${JSON.stringify(baseUrl)},\n  clientKey: ${JSON.stringify(sdkKey)},\n})\n\nfunction Feature() {\n  const enabled = useBooleanFlag('your-flag-key', false, {\n    targetingKey: 'user-123',\n  })\n  return enabled ? <NewExperience /> : <CurrentExperience />\n}\n\nroot.render(\n  <SwitchOnYourCodeProvider client={flags}>\n    <Feature />\n  </SwitchOnYourCodeProvider>,\n)`,
    },
  ]
}
