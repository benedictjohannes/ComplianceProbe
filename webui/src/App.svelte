<script lang="ts">
  import { onMount } from 'svelte';
  import Shield from 'lucide-svelte/icons/shield';
  import UploadCloud from 'lucide-svelte/icons/upload-cloud';
  import Folder from 'lucide-svelte/icons/folder';
  import Globe from 'lucide-svelte/icons/globe';
  import Play from 'lucide-svelte/icons/play';
  import RotateCcw from 'lucide-svelte/icons/rotate-ccw';
  import X from 'lucide-svelte/icons/x';
  import Lock from 'lucide-svelte/icons/lock';
  import CheckCircle2 from 'lucide-svelte/icons/check-circle-2';
  import XCircle from 'lucide-svelte/icons/x-circle';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Clock from 'lucide-svelte/icons/clock';
  import Terminal from 'lucide-svelte/icons/terminal';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Search from 'lucide-svelte/icons/search';
  import Download from 'lucide-svelte/icons/download';
  import FileText from 'lucide-svelte/icons/file-text';
  import {
    Button,
    Badge,
    Card,
    Dialog,
    Dropdown,
    Accordion,
    AccordionItem,
    Input,
    Switch,
    Tooltip,
  } from '$lib/components/ui';
  import {
    Header,
    ConnectionStatus,
    ThemeToggle,
    ErrorBanner,
    ShutdownModal,
  } from '$lib/components/common';
  import { themeStore } from '$lib/state/theme.svelte';

  let activeStep = $state<number>(1);
  let maxAccessibleStep = $state<number>(4);
  let playbookName = $state<string>('cis-ubuntu-22.04.yaml');
  let connectionState = $state<'connected' | 'reconnecting' | 'disconnected'>('connected');

  // Interactive demo dialog state
  let sampleDialogOpen = $state(false);
  let sampleSwitchChecked = $state(true);
  let sampleInputText = $state('https://compliance.internal/repo/playbook.yaml');

  onMount(() => {
    themeStore.init();
  });
</script>

<div class="min-h-screen flex flex-col bg-zinc-950 text-zinc-100 dark:bg-zinc-950 dark:text-zinc-100 font-sans selection:bg-sky-500/20 selection:text-sky-300">
  <!-- Persistent Global Shell Header -->
  <Header
    {activeStep}
    playbookName={activeStep > 1 ? playbookName : undefined}
    {connectionState}
    {maxAccessibleStep}
    onstepchange={(step) => {
      activeStep = step;
    }}
  />

  <!-- Main Shell Container -->
  <main class="flex-1 max-w-5xl w-full mx-auto px-4 py-6 sm:px-6 space-y-6">
    <!-- Dev Phase Step Switcher Bar -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/60 p-3 flex flex-wrap items-center justify-between gap-3 text-xs">
      <div class="flex items-center gap-2">
        <span class="text-zinc-400 font-medium">Pipeline Preview:</span>
        <div class="inline-flex rounded-md border border-zinc-800 bg-zinc-950 p-0.5">
          {#each [1, 2, 3, 4] as s}
            <button
              type="button"
              onclick={() => (activeStep = s)}
              class="px-2.5 py-1 rounded text-xs font-medium cursor-pointer transition-colors {activeStep === s ? 'bg-sky-500/20 text-sky-400 font-semibold' : 'text-zinc-400 hover:text-zinc-200'}"
            >
              Step {s}: {s === 1 ? 'Load' : s === 2 ? 'Inspect' : s === 3 ? 'Execute' : 'Results'}
            </button>
          {/each}
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span class="text-zinc-400">Connection:</span>
        <select
          bind:value={connectionState}
          class="bg-zinc-950 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-300 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-500 cursor-pointer"
        >
          <option value="connected">Connected (Green)</option>
          <option value="reconnecting">Reconnecting (Yellow)</option>
          <option value="disconnected">Disconnected (Red)</option>
        </select>
      </div>
    </div>

    <!-- STEP 1: LOAD VIEW (IDLE) PREVIEW -->
    {#if activeStep === 1}
      <div class="space-y-6 animate-in fade-in-50 duration-200">
        <div class="text-center space-y-1.5 pt-4">
          <h2 class="text-2xl font-bold tracking-tight text-zinc-100">Ingest Compliance Playbook</h2>
          <p class="text-sm text-zinc-400 max-w-md mx-auto">
            Upload or fetch a YAML or JSON compliance definition to inspect assertions and execute audits.
          </p>
        </div>

        <!-- Dropzone Container -->
        <div class="max-w-xl mx-auto rounded-xl border border-dashed border-zinc-700 bg-zinc-900/40 p-8 text-center space-y-4 hover:border-zinc-500 transition-colors">
          <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-zinc-800/80 text-zinc-400">
            <UploadCloud class="h-6 w-6" />
          </div>

          <div class="space-y-1">
            <div class="text-sm font-semibold text-zinc-200">Drag & drop your compliance playbook</div>
            <div class="text-xs font-mono text-zinc-500">Supports .yaml, .yml, and .json files</div>
          </div>

          <div class="pt-2">
            <Button
              variant="secondary"
              size="sm"
              onclick={() => {
                activeStep = 2;
              }}
            >
              <Folder class="h-3.5 w-3.5 mr-1.5 text-zinc-400" />
              Browse Local Files
            </Button>
          </div>
        </div>

        <!-- Divider -->
        <div class="max-w-xl mx-auto flex items-center gap-3">
          <div class="flex-1 h-px bg-zinc-800"></div>
          <span class="text-[11px] font-mono uppercase text-zinc-500">or</span>
          <div class="flex-1 h-px bg-zinc-800"></div>
        </div>

        <!-- Remote URL Action -->
        <div class="text-center">
          <Button
            variant="outline"
            size="sm"
            onclick={() => {
              sampleDialogOpen = true;
            }}
          >
            <Globe class="h-3.5 w-3.5 mr-1.5 text-sky-400" />
            Fetch from HTTPS URL...
          </Button>
        </div>

        <!-- Error Banner Example -->
        <div class="max-w-xl mx-auto pt-4">
          <ErrorBanner
            code="PLAYBOOK_VALIDATION_FAILED"
            message="Playbook contains validation errors"
            detail={[
              { path: 'sections[0].assertions[1].code', message: 'Duplicate assertion code "SEC-001"' },
              { path: 'report_destination.https.url', message: 'HTTPS URL is required when HTTPS reporting is enabled' },
            ]}
            onretry={() => {}}
            ondismiss={() => {}}
          />
        </div>
      </div>
    {/if}

    <!-- STEP 2: INSPECT VIEW (LOADED) PREVIEW -->
    {#if activeStep === 2}
      <div class="space-y-6 animate-in fade-in-50 duration-200">
        <!-- Playbook Header Card -->
        <Card class="space-y-3">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <Shield class="h-5 w-5 text-sky-400 shrink-0" />
                <h2 class="text-lg font-bold text-zinc-100">CIS Ubuntu 22.04 LTS Benchmark</h2>
              </div>
              <p class="text-sm text-zinc-400">
                Comprehensive security configuration benchmark for Ubuntu LTS server installations.
              </p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="warning" size="sm">
                <Lock class="h-3 w-3 mr-1" />
                Requires Sudo
              </Badge>
              <Badge variant="neutral" size="sm">4 Assertions</Badge>
              <Badge variant="default" size="sm">v1.2.0</Badge>
            </div>
          </div>

          <!-- Frontmatter Metadata Badges -->
          <div class="flex flex-wrap gap-2 pt-2 border-t border-zinc-800/80 text-xs font-mono text-zinc-400">
            <span class="bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">os: ubuntu-22.04</span>
            <span class="bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">profile: server-hardened</span>
            <span class="bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">author: sec-ops</span>
          </div>
        </Card>

        <!-- Destination Summary Cards -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Card class="flex items-center justify-between p-3.5">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-zinc-800 text-zinc-300">
                <Folder class="h-4 w-4" />
              </div>
              <div>
                <div class="text-xs text-zinc-400 font-medium">Local Report Folder</div>
                <div class="text-xs font-mono text-zinc-200">~/.local/state/crobe/runs/</div>
              </div>
            </div>
            <Badge variant="neutral" size="sm">Default</Badge>
          </Card>

          <Card class="flex items-center justify-between p-3.5">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-sky-500/10 text-sky-400">
                <Globe class="h-4 w-4" />
              </div>
              <div>
                <div class="text-xs text-zinc-400 font-medium">Remote HTTPS Delivery</div>
                <div class="text-xs font-mono text-sky-400">https://sec-ops.corp/ingest</div>
              </div>
            </div>
            <Badge variant="info" size="sm">Configured</Badge>
          </Card>
        </div>

        <!-- Section Accordion Tree -->
        <Card class="p-0 overflow-hidden">
          <div class="px-4 py-3 border-b border-zinc-800 bg-zinc-900/40 flex items-center justify-between">
            <h3 class="text-sm font-semibold text-zinc-200">Playbook Structure (2 Sections, 4 Assertions)</h3>
            <span class="text-xs text-zinc-500 font-mono">Progressive Disclosure</span>
          </div>

          <Accordion type="multiple" value={['sec-1', 'sec-2']} class="p-2">
            <AccordionItem value="sec-1" title="1. System Storage & Partition Isolation">
              {#snippet badge()}
                <Badge variant="neutral" size="sm">2 assertions</Badge>
              {/snippet}
              <div class="space-y-2 pt-2">
                <div class="rounded-lg border border-zinc-800 bg-zinc-950/60 p-3 space-y-2">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Badge variant="code" size="sm">SEC-001</Badge>
                      <span class="text-xs font-medium text-zinc-200">Separate /tmp partition is mounted</span>
                    </div>
                    <Badge variant="warning" size="sm">
                      <Lock class="h-2.5 w-2.5 mr-0.5" />
                      sudo
                    </Badge>
                  </div>
                  <p class="text-xs text-zinc-400">Verifies that the /tmp directory has its own dedicated mount point.</p>
                </div>

                <div class="rounded-lg border border-zinc-800 bg-zinc-950/60 p-3 space-y-2">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Badge variant="code" size="sm">SEC-002</Badge>
                      <span class="text-xs font-medium text-zinc-200">nodev & noexec options set on /dev/shm</span>
                    </div>
                    <Badge variant="warning" size="sm">
                      <Lock class="h-2.5 w-2.5 mr-0.5" />
                      sudo
                    </Badge>
                  </div>
                  <p class="text-xs text-zinc-400">Ensure shared memory space has execution prevention flags configured.</p>
                </div>
              </div>
            </AccordionItem>

            <AccordionItem value="sec-2" title="2. User Accounts & Access Controls">
              {#snippet badge()}
                <Badge variant="neutral" size="sm">2 assertions</Badge>
              {/snippet}
              <div class="space-y-2 pt-2">
                <div class="rounded-lg border border-zinc-800 bg-zinc-950/60 p-3 space-y-2">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Badge variant="code" size="sm">SEC-003</Badge>
                      <span class="text-xs font-medium text-zinc-200">Ensure root login over SSH is disabled</span>
                    </div>
                  </div>
                  <p class="text-xs text-zinc-400">Restricts direct root login capability over remote SSH connections.</p>
                </div>
              </div>
            </AccordionItem>
          </Accordion>
        </Card>

        <!-- Bottom Action Bar -->
        <div class="flex items-center justify-between pt-2">
          <Button
            variant="ghost"
            size="sm"
            onclick={() => {
              activeStep = 1;
            }}
            class="text-zinc-400 hover:text-rose-400"
          >
            <X class="h-4 w-4 mr-1.5" />
            Unload Playbook
          </Button>

          <Button
            variant="primary"
            size="md"
            onclick={() => {
              activeStep = 3;
            }}
          >
            <Play class="h-4 w-4 mr-1.5 fill-current" />
            Run Playbook
          </Button>
        </div>
      </div>
    {/if}

    <!-- STEP 3: EXECUTE VIEW (RUNNING) PREVIEW -->
    {#if activeStep === 3}
      <div class="space-y-6 animate-in fade-in-50 duration-200">
        <!-- Live Progress Metric Card -->
        <Card class="space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3 text-xs font-mono">
            <div class="flex items-center gap-2 text-zinc-200">
              <Clock class="h-4 w-4 text-sky-400" />
              <span class="font-semibold text-sm">00:08.4s</span>
              <span class="text-zinc-500">•</span>
              <span class="text-sky-400 font-semibold">60% (3/5 assertions)</span>
            </div>

            <div class="flex items-center gap-2">
              <Badge variant="passed" size="sm">✓ 2 Passed</Badge>
              <Badge variant="failed" size="sm">✕ 1 Failed</Badge>
              <Badge variant="running" size="sm">
                <Loader2 class="h-3 w-3 animate-spin mr-1" />
                1 Running
              </Badge>
            </div>

            <Button
              variant="destructive"
              size="xs"
              onclick={() => {
                activeStep = 2;
              }}
            >
              <X class="h-3.5 w-3.5 mr-1" />
              Cancel Run
            </Button>
          </div>

          <!-- Progress Bar Track -->
          <div class="h-2 w-full rounded-full bg-zinc-800 overflow-hidden">
            <div class="h-full bg-sky-500 transition-all duration-300" style="width: 60%"></div>
          </div>
        </Card>

        <!-- Live Assertion Stream Cards -->
        <div class="space-y-3">
          <!-- Passed Item -->
          <Card class="p-3 border-emerald-500/20 bg-emerald-500/5">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2.5">
                <CheckCircle2 class="h-4 w-4 text-emerald-400 shrink-0" />
                <Badge variant="code" size="sm">SEC-001</Badge>
                <span class="text-xs font-semibold text-zinc-100">Separate /tmp partition is mounted</span>
              </div>
              <div class="flex items-center gap-2 text-xs font-mono text-zinc-400">
                <span class="text-emerald-400">+1 pt</span>
                <span>18ms</span>
              </div>
            </div>
          </Card>

          <!-- Failed Item -->
          <Card class="p-3 border-rose-500/30 bg-rose-500/5 space-y-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2.5">
                <XCircle class="h-4 w-4 text-rose-400 shrink-0" />
                <Badge variant="code" size="sm">SEC-002</Badge>
                <span class="text-xs font-semibold text-zinc-100">nodev & noexec options set on /dev/shm</span>
              </div>
              <div class="flex items-center gap-2 text-xs font-mono text-zinc-400">
                <span class="text-rose-400">-1 pt</span>
                <span>45ms</span>
              </div>
            </div>

            <div class="rounded bg-rose-950/40 p-2.5 border border-rose-500/20 text-xs font-mono text-rose-300">
              <div class="font-semibold text-rose-200 mb-1">🚨 Failed Rules:</div>
              <div>• StdOutRule regex `/nodev/` failed to match output</div>
              <div class="mt-1 text-zinc-400">Command: findmnt -n /dev/shm (exit: 0)</div>
            </div>
          </Card>

          <!-- In-Flight Running Item -->
          <Card class="p-3 border-sky-500/40 bg-sky-500/5 space-y-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2.5">
                <Loader2 class="h-4 w-4 text-sky-400 animate-spin shrink-0" />
                <Badge variant="code" size="sm">SEC-003</Badge>
                <span class="text-xs font-semibold text-zinc-100">Disable automounting of USB devices</span>
              </div>
              <Badge variant="running" size="sm">00:02s</Badge>
            </div>
            <div class="text-xs font-mono text-sky-300 pl-6.5">
              ▶ Running Command [1/1]: modprobe -n -v usb-storage
            </div>
          </Card>
        </div>

        <!-- Terminal Log Preview Drawer -->
        <Card class="p-0 overflow-hidden border-zinc-800 bg-zinc-950">
          <div class="px-3 py-2 bg-zinc-900/80 border-b border-zinc-800 flex items-center justify-between text-xs font-mono">
            <div class="flex items-center gap-2 text-zinc-300">
              <Terminal class="h-3.5 w-3.5 text-sky-400" />
              <span>Console Logs (6 lines)</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-zinc-500">Auto-scroll: ON</span>
              <Button
                variant="primary"
                size="xs"
                onclick={() => {
                  activeStep = 4;
                }}
              >
                Simulate Finish ➜
              </Button>
            </div>
          </div>
          <div class="p-3 font-mono text-xs text-zinc-400 space-y-1 bg-zinc-950">
            <div>[21:19:01] <span class="text-sky-400">INFO</span> Starting playbook execution: CIS Ubuntu 22.04</div>
            <div>[21:19:02] <span class="text-sky-400">INFO</span> [SEC-001] Separate /tmp partition is mounted: <span class="text-emerald-400">PASSED</span> (18ms)</div>
            <div>[21:19:03] <span class="text-rose-400">FAIL</span> [SEC-002] nodev & noexec options set on /dev/shm: <span class="text-rose-400">FAILED</span> (45ms)</div>
            <div>[21:19:04] <span class="text-sky-400">INFO</span> [SEC-003] Disable automounting of USB devices: RUNNING...</div>
          </div>
        </Card>
      </div>
    {/if}

    <!-- STEP 4: RESULTS VIEW (COMPLETED) PREVIEW -->
    {#if activeStep === 4}
      <div class="space-y-6 animate-in fade-in-50 duration-200">
        <!-- Top Action Bar -->
        <div class="flex items-center justify-between">
          <Button
            variant="outline"
            size="sm"
            onclick={() => {
              activeStep = 3;
            }}
          >
            <RotateCcw class="h-3.5 w-3.5 mr-1.5" />
            Re-run Playbook
          </Button>

          <Button
            variant="outline"
            size="sm"
            onclick={() => {
              activeStep = 1;
            }}
          >
            <Folder class="h-3.5 w-3.5 mr-1.5" />
            Load Another Playbook
          </Button>
        </div>

        <!-- Scorecard Hero -->
        <Card class="p-6 space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-4">
              <!-- Radial Gauge Pill -->
              <div class="flex h-16 w-16 items-center justify-center rounded-full border-4 border-emerald-500 bg-emerald-500/10 text-emerald-400 font-bold text-xl font-mono">
                85%
              </div>
              <div class="space-y-1">
                <div class="flex items-center gap-2">
                  <Badge variant="passed" size="sm" class="font-bold">PASSED</Badge>
                  <span class="text-xs text-zinc-400 font-mono">(17/20 Assertions)</span>
                </div>
                <h3 class="text-base font-bold text-zinc-100">{playbookName}</h3>
                <p class="text-xs font-mono text-zinc-400">Target: root@ubuntu-prod (linux/amd64) • Duration: 01.42s</p>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <div class="text-center px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800">
                <div class="text-base font-bold text-emerald-400 font-mono">17</div>
                <div class="text-[10px] text-zinc-500 uppercase">Passed</div>
              </div>
              <div class="text-center px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800">
                <div class="text-base font-bold text-rose-400 font-mono">3</div>
                <div class="text-[10px] text-zinc-500 uppercase">Failed</div>
              </div>
              <div class="text-center px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800">
                <div class="text-base font-bold text-zinc-200 font-mono">20</div>
                <div class="text-[10px] text-zinc-500 uppercase">Total</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Bottom Reports Action Bar -->
        <div class="flex items-center justify-between pt-2">
          <Dropdown
            side="top"
            align="start"
            items={[
              { id: 'md', label: 'Markdown Report (.md)', icon: FileText },
              { id: 'log', label: 'Execution Logs (.log)', icon: Terminal },
              { id: 'sep1', label: '', separator: true },
              { id: 'zip', label: 'Download Archive (.zip)', icon: Download },
              { id: 'tar', label: 'Download Tarball (.tar.gz)', icon: Download },
            ]}
          >
            {#snippet trigger()}
              <Button variant="secondary" size="md">
                <FileText class="h-4 w-4 mr-1.5" />
                Reports & Downloads ▴
              </Button>
            {/snippet}
          </Dropdown>

          <Button variant="indigo" size="md">
            <Globe class="h-4 w-4 mr-1.5" />
            Submit Report to Server
          </Button>
        </div>
      </div>
    {/if}

    <!-- UI PRIMITIVES SHOWCASE (COLLAPSIBLE TEST SECTION) -->
    <div class="pt-8 border-t border-zinc-800">
      <Card class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h4 class="text-sm font-bold text-zinc-200">Phase 3A: Design Primitives Verification & Showcase</h4>
            <p class="text-xs text-zinc-400">Interactive testbed for Button, Badge, Card, Dialog, Dropdown, Accordion, Input, Switch, and Tooltip.</p>
          </div>
          <Badge variant="passed" size="sm">Phase 3A Active</Badge>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 pt-2">
          <!-- Button Variants -->
          <div class="space-y-2 p-3 rounded bg-zinc-950 border border-zinc-800 text-xs">
            <div class="font-semibold text-zinc-300">Buttons:</div>
            <div class="flex flex-wrap gap-1.5">
              <Button variant="primary" size="xs">Primary</Button>
              <Button variant="secondary" size="xs">Secondary</Button>
              <Button variant="outline" size="xs">Outline</Button>
              <Button variant="destructive" size="xs">Destructive</Button>
              <Button variant="accent" size="xs">Accent</Button>
            </div>
          </div>

          <!-- Badges -->
          <div class="space-y-2 p-3 rounded bg-zinc-950 border border-zinc-800 text-xs">
            <div class="font-semibold text-zinc-300">Badges:</div>
            <div class="flex flex-wrap gap-1.5">
              <Badge variant="passed" size="sm">Passed</Badge>
              <Badge variant="failed" size="sm">Failed</Badge>
              <Badge variant="warning" size="sm">Warning</Badge>
              <Badge variant="running" size="sm">Running</Badge>
              <Badge variant="code" size="sm">SEC-001</Badge>
            </div>
          </div>

          <!-- Controls: Switch, Input, Tooltip -->
          <div class="space-y-2 p-3 rounded bg-zinc-950 border border-zinc-800 text-xs">
            <div class="font-semibold text-zinc-300">Controls & Tooltips:</div>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <Switch bind:checked={sampleSwitchChecked} />
                <span class="text-zinc-400">{sampleSwitchChecked ? 'Enabled' : 'Disabled'}</span>
              </div>
              <Tooltip content="Clean tooltip preview with high contrast!">
                <Badge variant="outline" size="sm" class="cursor-help">Hover for Tooltip</Badge>
              </Tooltip>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </main>

  <!-- Sample Remote URL Fetch Modal Dialog -->
  <Dialog
    bind:open={sampleDialogOpen}
    title="Fetch Remote Playbook"
    description="Enter an HTTPS endpoint serving a valid YAML or JSON playbook definition."
  >
    <div class="space-y-3 py-1">
      <div class="space-y-1">
        <label for="url-input" class="text-xs font-medium text-zinc-300">Playbook HTTPS URL</label>
        <Input
          id="url-input"
          bind:value={sampleInputText}
          mono
          placeholder="https://example.com/playbook.yaml"
        >
          {#snippet leading()}
            <Globe class="h-4 w-4" />
          {/snippet}
        </Input>
      </div>
    </div>

    {#snippet footer()}
      <Button variant="ghost" size="sm" onclick={() => (sampleDialogOpen = false)}>
        Cancel
      </Button>
      <Button variant="primary" size="sm" onclick={() => (sampleDialogOpen = false)}>
        Fetch & Load ➜
      </Button>
    {/snippet}
  </Dialog>
</div>
