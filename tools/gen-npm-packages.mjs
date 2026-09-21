// @ts-check
/// <reference lib="es2023" />

import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const repoRoot = path.join(import.meta.dirname, '..');

// The published version is the one committed on the branch, never an input
const npmPackageVersion = (await fs.readFile(path.join(repoRoot, 'VERSION'), 'utf8')).trim();
assert.match(npmPackageVersion, /^\d+\.\d+\.\d+$/, 'VERSION must hold a plain semver');
if (process.env.TSGOLINT_VERSION != null) {
  assert.equal(process.env.TSGOLINT_VERSION, npmPackageVersion, 'TSGOLINT_VERSION differs from VERSION');
}

const MAIN_PACKAGE = '@block65/oxlint-tsgolint';

const GOOS2PROCESS_PLATFORM = {
  windows: 'win32',
  linux: 'linux',
  darwin: 'darwin',
};
const GOARCH2PROCESS_ARCH = {
  amd64: 'x64',
  arm64: 'arm64',
};

// Linux only, the platforms Block65 runs; extend the release workflow matrix alongside
const BUILT = [
  ['linux', 'amd64'],
  ['linux', 'arm64'],
];

const binariesMatrix = BUILT.map(([goos, goarch]) => {
  const platform = GOOS2PROCESS_PLATFORM[goos];
  const arch = GOARCH2PROCESS_ARCH[goarch];
  return {
    goarch,
    goos,
    arch,
    platform,
    artifactName: `tsgolint-${goos}-${goarch}`,
    npmPackageName: `${MAIN_PACKAGE}-${platform}-${arch}`,
  };
});

const commonPackageJson = {
  version: npmPackageVersion,
  description:
    'Block65 build of tsgolint 7.0.2002 with additional type-aware rules. Internal use.',
  license: 'MIT',
  author: 'Block65',
  repository: 'github:block65/tsgolint',
  // npm fills bugs in from repository when it is absent, and issues are off
  // on this repo, so an explicit value keeps the registry off a dead link
  bugs: { url: 'https://github.com/block65/tsgolint#readme' },
  publishConfig: {
    access: 'public',
  },
};

const npmDir = path.join(repoRoot, 'npm');
const licensePath = path.join(repoRoot, 'LICENSE');
const noticePath = path.join(repoRoot, 'NOTICE');
const readmePath = path.join(repoRoot, 'README.md');
const buildDir = path.join(repoRoot, 'build');

await Promise.all([
  ...binariesMatrix.map(
    async ({ arch, platform, artifactName, npmPackageName }) => {
      const packageName = `${platform}-${arch}`;
      const packageDir = path.join(npmDir, packageName);
      const binaryName = `tsgolint${platform === 'win32' ? '.exe' : ''}`;

      await fs.rm(packageDir, { recursive: true, force: true });
      await fs.mkdir(packageDir);
      await Promise.all([
        fs.writeFile(
          path.join(packageDir, 'package.json'),
          JSON.stringify(
            {
              ...commonPackageJson,
              publishConfig: {
                ...commonPackageJson.publishConfig,
                executableFiles: [binaryName],
              },
              name: npmPackageName,
              preferUnplugged: true,
              files: [binaryName, 'NOTICE'],
              os: [platform],
              cpu: [arch],
            },
            null,
            2,
          ),
        ),
        fs.copyFile(licensePath, path.join(packageDir, 'LICENSE')),
        fs.copyFile(noticePath, path.join(packageDir, 'NOTICE')),
        fs.copyFile(
          path.join(buildDir, artifactName, 'tsgolint'),
          path.join(packageDir, binaryName),
        ),
      ]);
    },
  ),
  (async () => {
    const packageDir = path.join(npmDir, 'core');
    await Promise.all([
      fs.writeFile(
        path.join(packageDir, 'package.json'),
        JSON.stringify(
          {
            ...commonPackageJson,
            name: MAIN_PACKAGE,
            bin: {
              tsgolint: './bin/tsgolint.js',
            },
            optionalDependencies: Object.fromEntries(
              binariesMatrix.map(({ npmPackageName }) => [
                npmPackageName,
                npmPackageVersion,
              ]),
            ),
          },
          null,
          2,
        ),
      ),
      fs.copyFile(licensePath, path.join(packageDir, 'LICENSE')),
      fs.copyFile(noticePath, path.join(packageDir, 'NOTICE')),
      fs.copyFile(readmePath, path.join(packageDir, 'README.md')),
    ]);
  })(),
]);

