import { spawn, ChildProcess } from 'node:child_process';
import path from 'node:path';
import fs from 'node:fs';

export interface RunningServer {
  url: string;
  baseUrl: string;
  token: string;
  port: number;
  process: ChildProcess;
  stop: () => Promise<void>;
}

export async function startTestServer(options: { playbookPath?: string; reportsFolder?: string } = {}): Promise<RunningServer> {
  const binaryPath = path.resolve(process.cwd(), '../crobe-linux');
  
  if (!fs.existsSync(binaryPath)) {
    throw new Error(`Binary not found at ${binaryPath}. Did you run 'make build-linux'?`);
  }

  const args = ['-ui', '-no-open', '-port', '0'];
  if (options.reportsFolder) {
    args.push('-folder', options.reportsFolder);
  }
  if (options.playbookPath) {
    args.push(options.playbookPath);
  }

  const proc = spawn(binaryPath, args, {
    cwd: path.resolve(process.cwd(), '..'),
    env: { ...process.env },
  });

  return new Promise((resolve, reject) => {
    let stdoutBuffer = '';
    let stderrBuffer = '';
    let resolved = false;

    const timeout = setTimeout(() => {
      if (!resolved) {
        proc.kill('SIGKILL');
        reject(new Error(`Server timed out starting up.\nStdout: ${stdoutBuffer}\nStderr: ${stderrBuffer}`));
      }
    }, 10000);

    const onData = (data: Buffer) => {
      const text = data.toString();
      stdoutBuffer += text;

      // Look for: http://127.0.0.1:12345/?token=abc123xyz
      const match = stdoutBuffer.match(/http:\/\/(127\.0\.0\.1|localhost):(\d+)\/\?token=([a-zA-Z0-9_-]+)/);
      if (match && !resolved) {
        resolved = true;
        clearTimeout(timeout);

        const url = match[0];
        const port = parseInt(match[2], 10);
        const token = match[3];
        const baseUrl = `http://${match[1]}:${port}`;

        resolve({
          url,
          baseUrl,
          token,
          port,
          process: proc,
          stop: async () => {
            return new Promise<void>((res) => {
              if (proc.killed || proc.exitCode !== null) {
                res();
                return;
              }
              proc.once('exit', () => res());
              proc.kill('SIGTERM');
              // Fallback force kill after 3s
              setTimeout(() => {
                try {
                  if (proc.exitCode === null) {
                    proc.kill('SIGKILL');
                  }
                } catch {
                  // ignore
                }
                res();
              }, 3000);
            });
          },
        });
      }
    };

    proc.stdout?.on('data', onData);
    proc.stderr?.on('data', (data: Buffer) => {
      stderrBuffer += data.toString();
    });

    proc.on('error', (err) => {
      if (!resolved) {
        clearTimeout(timeout);
        reject(err);
      }
    });

    proc.on('exit', (code) => {
      if (!resolved) {
        clearTimeout(timeout);
        reject(new Error(`Server exited prematurely with code ${code}.\nStdout: ${stdoutBuffer}\nStderr: ${stderrBuffer}`));
      }
    });
  });
}
