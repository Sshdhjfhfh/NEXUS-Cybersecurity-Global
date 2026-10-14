#!/usr/bin/env bash
# Nexus IA v6.0 — Script de traducción del frontend
#
# Reemplaza los strings chinos más comunes por claves i18n.
# NO reemplaza todo (algunos strings son contextuales), solo los
# que tienen equivalente directo en las traducciones.
#
# Uso:
#   ./scripts/translate_frontend.sh          # dry-run (ver qué cambiaría)
#   ./scripts/translate_frontend.sh --apply  # aplicar cambios

set -uo pipefail

WEB_DIR="web/src"
APPLY=0
[ "${1:-}" = "--apply" ] && APPLY=1

if [ ! -d "$WEB_DIR" ]; then
    echo "❌ Directorio $WEB_DIR no encontrado"
    exit 1
fi

# Colores
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

# Diccionario de reemplazos (chino → clave i18n)
# Formato: "chino|clave"
REPLACEMENTS=(
    # Navegación
    "仪表盘|nav.dashboard"
    "任务|nav.tasks"
    "资产|nav.assets"
    "漏洞|nav.findings"
    "流量|nav.traffic"
    "智能体|nav.agents"
    "技能|nav.skills"
    "工具|nav.tools"
    "通知|nav.notifications"
    "拦截|nav.intercept"
    "审批|nav.approvals"
    "设置|nav.settings"
    "日志|nav.logs"
    "工作空间|nav.workspace"
    "对话|nav.chat"
    "系统|nav.system"
    "功能|nav.function"
    "同步|nav.sync"
    "命令|nav.commands"

    # Comunes
    "保存|common.save"
    "取消|common.cancel"
    "删除|common.delete"
    "编辑|common.edit"
    "添加|common.add"
    "创建|common.create"
    "更新|common.update"
    "刷新|common.refresh"
    "搜索|common.search"
    "过滤|common.filter"
    "导出|common.export"
    "导入|common.import"
    "确认|common.confirm"
    "返回|common.back"
    "关闭|common.close"
    "打开|common.open"
    "详情|common.details"
    "加载中|common.loading"
    "暂无数据|common.no_data"
    "没有结果|common.no_results"
    "全部|common.all"
    "名称|common.name"
    "描述|common.description"
    "状态|common.status"
    "时间|common.time"
    "创建时间|common.created_at"
    "更新时间|common.updated_at"
    "操作|common.actions"

    # Auth
    "登录|auth.login"
    "登出|auth.logout"
    "用户名|auth.username"
    "密码|auth.password"
    "账户|auth.account"
    "修改密码|auth.change_password"
    "初始化密码|auth.init_password"
    "新密码|auth.new_password"
    "确认密码|auth.confirm_password"

    # Estados
    "进行中|status.in_progress"
    "已完成|status.completed"
    "失败|status.failed"
    "待处理|status.pending"
    "暂停|status.paused"
    "运行中|status.running"
    "排队中|status.queued"
    "已停止|status.stopped"

    # Mensajes
    "操作成功|messages.operation_success"
    "操作失败|messages.operation_failed"
    "保存成功|messages.saved_success"
    "保存失败|messages.saved_failed"
    "删除成功|messages.deleted_success"
    "删除失败|messages.deleted_failed"
    "网络错误|messages.network_error"
    "服务器错误|messages.server_error"

    # Tareas
    "任务列表|task.list"
    "新建任务|task.new"
    "任务详情|task.details"
    "任务名称|task.name"
    "任务描述|task.description"
    "任务状态|task.status"

    # Hallazgos
    "漏洞列表|finding.list"
    "漏洞详情|finding.details"
    "严重|finding.critical"
    "高危|finding.high"
    "中危|finding.medium"
    "低危|finding.low"
    "已修复|finding.fixed"
    "未修复|finding.unfixed"
)

# Función: contar ocurrencias de un string chino en el proyecto
count_occurrences() {
    local chinese="$1"
    local count
    count=$(grep -rln "$chinese" "$WEB_DIR" --include="*.ts" --include="*.tsx" 2>/dev/null | wc -l) || true
    echo "${count:-0}"
}

# Función: aplicar reemplazo
apply_replacement() {
    local chinese="$1"
    local key="$2"

    # Buscar en strings literales (comillas dobles o simples)
    # Reemplaza: "chino" → t("key")
    find "$WEB_DIR" -name "*.tsx" -o -name "*.ts" | while read -r file; do
        # "chino" → t('key')
        sed -i "s|\"${chinese}\"|t(\"${key}\")|g" "$file"
        # 'chino' → t('key')
        sed -i "s|'${chinese}'|t(\"${key}\")|g" "$file"
        # >chino< → >{t("key")}<
        sed -i "s|>${chinese}<|>{t(\"${key}\")}<|g" "$file"
        # placeholder="chino" → placeholder={t("key")}
        sed -i "s|placeholder=\"${chinese}\"|placeholder={t(\"${key}\")}|g" "$file"
        # title="chino" → title={t("key")}
        sed -i "s|title=\"${chinese}\"|title={t(\"${key}\")}|g" "$file"
        # aria-label="chino" → aria-label={t("key")}
        sed -i "s|aria-label=\"${chinese}\"|aria-label={t(\"${key}\")}|g" "$file"
        # label="chino" → label={t("key")}
        sed -i "s|label=\"${chinese}\"|label={t(\"${key}\")}|g" "$file"
    done
}

# Función: añadir import de i18n a un archivo (si no lo tiene)
ensure_import() {
    local file="$1"
    if grep -q "from '@/lib/i18n'" "$file" 2>/dev/null; then
        return
    fi
    if grep -q 'from "@/lib/i18n"' "$file" 2>/dev/null; then
        return
    fi

    # Añadir import después de 'use client';
    if grep -q "^'use client'" "$file" 2>/dev/null; then
        sed -i "/^'use client'/a\\
\\
import { useTranslation } from '@/lib/i18n';" "$file"
    fi
}

echo "═══════════════════════════════════════"
echo "  Traducción del frontend"
echo "═══════════════════════════════════════"
[ "$APPLY" = "0" ] && echo -e "${YELLOW}DRY RUN — usa --apply para aplicar${NC}"
echo ""

total_changes=0
for entry in "${REPLACEMENTS[@]}"; do
    chinese="${entry%%|*}"
    key="${entry##*|}"

    count=$(count_occurrences "$chinese")
    if [ "$count" -gt 0 ]; then
        printf "  ${GREEN}%3d archivos${NC}  %-20s → %s\n" "$count" "$chinese" "$key"
        total_changes=$((total_changes + count))

        if [ "$APPLY" = "1" ]; then
            apply_replacement "$chinese" "$key"
        fi
    fi
done

echo ""
echo "  ─────────────────────────────────────────"
echo "  Total de archivos afectados: $total_changes"
echo ""

if [ "$APPLY" = "1" ]; then
    echo -e "${GREEN}✅ Cambios aplicados${NC}"
    echo ""
    echo "  Siguiente paso: añadir el hook useTranslation a cada componente."
    echo "  Ejemplo:"
    echo "    const { t } = useTranslation();"
    echo "    <h1>{t('nav.dashboard')}</h1>"
else
    echo -e "${YELLOW}Modo dry-run. Para aplicar:${NC}"
    echo "  ./scripts/translate_frontend.sh --apply"
fi
