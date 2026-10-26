"""Install the nexus command in the user's existing Go bin directory on PATH."""
import os
from pathlib import Path
import sys


def install():
    source = Path(__file__).resolve().parent / "nexus"
    directory = Path.home() / "go" / "bin"
    destination = directory / "nexus"
    if str(directory) not in os.environ.get("PATH", "").split(os.pathsep):
        raise ValueError(str(directory) + " no está en PATH; no se modificó el perfil de tu shell")
    directory.mkdir(parents=True, exist_ok=True)
    if destination.is_symlink() and destination.resolve() == source:
        print("nexus ya está instalado: " + str(destination))
        return
    if destination.exists() or destination.is_symlink():
        raise ValueError("Ya existe otro comando nexus en " + str(destination))
    destination.symlink_to(source)
    print("Instalado: nexus → " + str(source))


if __name__ == "__main__":
    try:
        install()
    except (OSError, ValueError) as exc:
        print("Error: " + str(exc), file=sys.stderr)
        sys.exit(1)
