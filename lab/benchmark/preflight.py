"""Check actual runtime dependencies inside the Nexus laboratory container."""
import json
import shutil
import subprocess
import sys


def main():
    names = ("bash", "python3", "rg", "curl", "git", "jq", "node", "npm", "nmap", "dig", "whois", "chromium")
    paths = {name: shutil.which(name) for name in names}
    checks = {"executables": paths, "missing": [name for name, path in paths.items() if not path]}
    if checks["missing"]:
        print(json.dumps(checks, indent=2))
        return 1
    commands = {
        "bash": ["bash", "-lc", "printf NEXUS_BASH_READY"],
        "python": ["python3", "-c", "print('NEXUS_PYTHON_READY')"],
        "browser": ["node", "-e", "(async()=>{const {chromium}=require('playwright');const b=await chromium.launch({headless:true});try{const p=await b.newPage();await p.setContent('<title>NEXUS_BROWSER_READY</title>');console.log(await p.title());}finally{await b.close();}})().catch(e=>{console.error(e.message);process.exit(1);})"],
    }
    checks["execution"] = {}
    for name, command in commands.items():
        try:
            result = subprocess.run(command, capture_output=True, text=True, timeout=60)
            checks["execution"][name] = {"returncode": result.returncode, "output": (result.stdout + result.stderr)[-2000:]}
        except subprocess.TimeoutExpired:
            checks["execution"][name] = {"returncode": -1, "output": "timeout"}
    print(json.dumps(checks, indent=2))
    return int(any(item["returncode"] != 0 for item in checks["execution"].values()))


if __name__ == "__main__":
    sys.exit(main())
