#!/usr/bin/env bash
# Nexus IA v6.0 — Añade imports de i18n a archivos que usan t()
#
# Detecta archivos .tsx/.ts con llamadas a t("...") y añade:
#   import { t } from '@/lib/i18n';
#
# Solo si no lo tienen ya.

set -uo pipefail

WEB_DIR="web/src"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

echo "═══════════════════════════════════════"
echo "  Añadir imports de i18n"
echo "═══════════════════════════════════════"

# Encontrar archivos que usan t("...")
files=$(grep -rln 't("' "$WEB_DIR" --include="*.tsx" --include="*.ts" 2>/dev/null || true)

if [ -z "$files" ]; then
    echo -e "${YELLOW}No se encontraron archivos con t(\"...\")${NC}"
    exit 0
fi

count=0
total=$(echo "$files" | wc -l)

echo "Procesando $total archivos..."
echo ""

while IFS= read -r file; do
    [ -z "$file" ] && continue

    # ¿Ya tiene import?
    if grep -q "from '@/lib/i18n'" "$file" 2>/dev/null; then
        continue
    fi
    if grep -q 'from "@/lib/i18n"' "$file" 2>/dev/null; then
        continue
    fi

    # ¿Ya tiene import de useTranslation?
    if grep -q "useTranslation" "$file" 2>/dev/null; then
        # Ya usa el hook, no necesita import de t
        continue
    fi

    # Añadir import después de 'use client'; o del primer import
    if grep -q "^'use client'" "$file" 2>/dev/null; then
        # Insertar después de 'use client'
        sed -i "0,/^'use client'/s||'use client';\n\nimport { t } from '@/lib/i18n';|" "$file"
    elif grep -q '^"use client"' "$file" 2>/dev/null; then
        sed -i '0,/^"use client"/s||"use client";\n\nimport { t } from "'"'"'@/lib/i18n'"'"';|' "$file"
    else
        # Insertar al principio
        sed -i "1i import { t } from '@/lib/i18n';\n" "$file"
    fi

    count=$((count + 1))
    printf "  ${GREEN}✅${NC} %s\n" "$file"
done <<< "$files"

echo ""
echo "  ─────────────────────────────────────────"
echo -e "  ${GREEN}Total: $count archivos modificados${NC}"
