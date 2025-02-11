// @ts-check

const { spawn } = require('child_process')

function build() {
  return spawn('yarn', ['run', 'build'], {
    stdio: 'inherit',
    shell: true,
    env: { ...process.env, NODE_OPTIONS: '--max_old_space_size=8192' },
  })
}

function watch() {
  return spawn('yarn', ['run', 'dev'], {
    stdio: 'inherit',
    shell: true,
    env: { ...process.env, NODE_OPTIONS: '--max_old_space_size=8192' },
  })
}

module.exports = { build, watch }
