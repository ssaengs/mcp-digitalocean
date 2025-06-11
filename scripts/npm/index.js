#!/usr/bin/env node

const path = require('path')
const fs = require('fs')
const childProcess = require('child_process')
const os = require('os')

/**
 * Run the platform-specific executable with the given arguments
 * @param {string[]} args Arguments to pass to the executable
 * @returns {Promise<number>} Returns a Promise that resolves to the exit code
 */
function runExecutable(args = []) {
    try {
        const packageJson = require('./package.json')

        // here we map the executable based on the platform.

        const platform = os.platform()
        const arch = os.arch()

        console.log('Detected platform:', platform)
        console.log('Detected architecture:', arch)

        const binKey = `mcp-digitalocean-${platform}-${arch}`;
        const execName = packageJson.bin[binKey]
        console.error('Found executable in package.json:', execName.toString())

        // The platform-specific executable should be in the same folder
        const execPath = path.join(__dirname, execName)
        console.log('Executable path:', execPath)

        if (!fs.existsSync(execPath)) {
            console.error(`Executable "${execPath}" not found.`)
            return Promise.resolve(1)
        }

        console.log(`Starting ${execPath}`) // Only logs in debug mode

        const child = childProcess.spawn(execPath, args, {
            stdio: 'inherit',
            shell: false
        })

        return new Promise((resolve) => {
            child.on('error', (err) => {
                console.error(`Error executing package: ${err.message}`)
                resolve(1)
            })

            child.on('exit', (code) => {
                resolve(code || 0)
            })
        })
    } catch (err) {
        console.error(`Error running executable: ${err.message}`)
        return Promise.resolve(1)
    }
}

// Check if this file is being run directly
if (require.main === module) {
    // Run the executable with command line args and exit with its code
    runExecutable(process.argv.slice(2))
        .then(code => process.exit(code))
} else {
    // Export the function for consumers to use
    module.exports = { runExecutable }
}