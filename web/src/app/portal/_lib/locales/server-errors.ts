// The portal API answers errors in Spanish (its source language). This table maps
// every message a portal client can receive to English, so an English UI never
// shows Spanish server text. Unknown messages pass through unchanged. Protocol
// sentinels the UI branches on ("mfa_required", "unauthorized") are deliberately
// NOT listed: they must reach the code untouched.

export const SERVER_ERRORS_EN: Record<string, string> = {
  "almacenamiento del portal no disponible": "The portal storage is unavailable",
  "almacenamiento no disponible": "Storage is unavailable",
  "contraseña incorrecta": "Incorrect password",
  "cuenta desactivada": "Account deactivated",
  "cuerpo inválido": "Invalid request",
  "código inválido": "Invalid code",
  "demasiados intentos fallidos; inténtalo más tarde": "Too many failed attempts; please try again later",
  "email inválido": "Invalid email",
  "email o contraseña incorrectos": "Incorrect email or password",
  "enlace inválido o expirado": "Invalid or expired link",
  "ese email ya está registrado": "That email is already registered",
  "ese email ya tiene una cuenta": "That email already has an account",
  "hallazgo no encontrado": "Finding not found",
  "id inválido": "Invalid id",
  "idioma no soportado": "Unsupported language",
  "invitación no válida o ya usada": "Invalid or already used invitation",
  "la contraseña actual es incorrecta": "The current password is incorrect",
  "la contraseña debe tener entre 8 y 72 caracteres": "The password must be between 8 and 72 characters",
  "la organización debe conservar al menos un propietario": "The organization must keep at least one owner",
  "la organización debe conservar al menos un propietario activo":
    "The organization must keep at least one active owner",
  "la solicitud no tiene una tarea vinculada todavía": "The request is not linked to an assessment yet",
  "miembro no encontrado": "Member not found",
  "no autorizado": "Not authorized",
  "no puedes desactivar tu propia cuenta": "You cannot deactivate your own account",
  "no se pudieron guardar las preferencias": "The preferences could not be saved",
  "no se pudieron listar las invitaciones": "The invitations could not be listed",
  "no se pudieron listar las solicitudes": "The requests could not be listed",
  "no se pudieron listar los miembros": "The members could not be listed",
  "no se pudieron obtener los resultados": "The results could not be retrieved",
  "no se pudieron revocar las sesiones": "The sessions could not be revoked",
  "no se pudo aceptar la invitación": "The invitation could not be accepted",
  "no se pudo activar": "Could not enable",
  "no se pudo actualizar": "Could not update",
  "no se pudo actualizar el rol": "The role could not be updated",
  "no se pudo actualizar la contraseña": "The password could not be updated",
  "no se pudo actualizar la cuenta": "The account could not be updated",
  "no se pudo calcular el resumen": "The overview could not be computed",
  "no se pudo crear la cuenta": "The account could not be created",
  "no se pudo crear la invitación": "The invitation could not be created",
  "no se pudo crear la solicitud": "The request could not be created",
  "no se pudo desactivar": "Could not disable",
  "no se pudo eliminar el miembro": "The member could not be removed",
  "no se pudo eliminar la cuenta": "The account could not be deleted",
  "no se pudo eliminar la organización": "The organization could not be deleted",
  "no se pudo emitir el token": "The session could not be issued",
  "no se pudo exportar": "The export failed",
  "no se pudo generar el secreto": "The secret could not be generated",
  "no se pudo generar la invitación": "The invitation could not be generated",
  "no se pudo guardar el secreto": "The secret could not be saved",
  "no se pudo leer la auditoría": "The audit log could not be read",
  "no se pudo obtener el hallazgo": "The finding could not be retrieved",
  "no se pudo revocar la invitación": "The invitation could not be revoked",
  "no se pudo verificar el email": "The email could not be verified",
  "no se pudo verificar la auditoría": "The audit log could not be verified",
  "no tienes permisos para esta acción": "You do not have permission for this action",
  "nombre de organización inválido": "Invalid organization name",
  "organización no encontrada": "Organization not found",
  "organización suspendida": "Organization suspended",
  "organización suspendida; contacta con soporte": "Organization suspended; please contact support",
  "perfil de vista inválido": "Invalid view profile",
  "primero ejecuta la configuración": "Please run the setup first",
  "resultado no disponible": "Result not available",
  "rol de invitación inválido": "Invalid invitation role",
  "rol inválido": "Invalid role",
  "rol inválido (admin, member o auditor)": "Invalid role (admin, member or auditor)",
  "el rol auditor es de solo lectura": "The auditor role is read-only",
  "secreto inválido": "Invalid secret",
  "sesión revocada; inicia sesión de nuevo": "Session revoked; please sign in again",
  "solicitud no encontrada": "Request not found",
  "token requerido": "Token required",
  "tu cuenta ha sido desactivada; contacta con un administrador":
    "Your account has been deactivated; please contact an administrator",
  "título inválido": "Invalid title",
  "usuario no encontrado": "User not found",
  // remediation workspace
  "no se pudo cargar la remediación": "Remediation could not be loaded",
  "responsable inválido": "Invalid owner",
  "el responsable debe ser un miembro activo de la organización":
    "The owner must be an active member of the organization",
  "fecha inválida": "Invalid date",
  "estado inválido": "Invalid state",
  "solo un administrador puede revocar una aceptación de riesgo": "Only an administrator can revoke a risk acceptance",
  "el hallazgo ya está corregido y verificado": "This finding is already fixed and verified",
  "el comentario debe tener entre 1 y 2000 caracteres": "Comments must be between 1 and 2000 characters",
  "no se pudo guardar el comentario": "The comment could not be saved",
  "la nota debe tener como máximo 1000 caracteres": "The note must be at most 1000 characters",
  "ya hay una verificación pendiente para este hallazgo": "A verification is already pending for this finding",
  "el riesgo está aceptado; revócalo antes de solicitar una verificación":
    "The risk is accepted; revoke the acceptance before requesting a verification",
  "no se pudo solicitar la verificación": "The verification could not be requested",
  "la justificación debe tener entre 10 y 2000 caracteres": "The justification must be between 10 and 2000 characters",
  "la aceptación debe caducar entre mañana y dentro de 365 días":
    "The acceptance must expire between tomorrow and 365 days from now",
  "no se pudo aceptar el riesgo": "The risk could not be accepted",
  "no hay una aceptación de riesgo que revocar": "There is no risk acceptance to revoke",
  // PDF reports
  "tipo de informe inválido": "Invalid report type",
  "no se pudo generar el informe": "The report could not be generated",
  // targets + rules of engagement
  "objetivo inválido": "Invalid target",
  "dirección IP no admitida": "IP address not allowed",
  "indica un dominio o IP concreto, sin comodines ni puertos":
    "Enter a specific domain or IP, without wildcards or ports",
  "ese objetivo ya está registrado": "That target is already registered",
  "has alcanzado el máximo de objetivos de tu organización":
    "Your organization has reached its maximum number of targets",
  "no se pudo registrar el objetivo": "The target could not be registered",
  "no se pudieron listar los objetivos": "The targets could not be listed",
  "no se pudo leer el objetivo": "The target could not be read",
  "objetivo no encontrado": "Target not found",
  "método de verificación inválido": "Invalid verification method",
  "NEXUS rechazó este objetivo; contacta con tu gestor":
    "NEXUS rejected this target; please contact your account manager",
  "la verificación por DNS solo aplica a dominios": "DNS verification only applies to domains",
  "espera unos segundos antes de volver a comprobar": "Please wait a few seconds before checking again",
  "no se pudo guardar la verificación": "The verification could not be saved",
  "el objetivo forma parte de una evaluación en curso": "The target is part of an assessment in progress",
  "no se pudo eliminar el objetivo": "The target could not be deleted",
  "la dirección no es pública; solicita la aprobación manual de NEXUS":
    "The address is not public; ask NEXUS for manual approval",
  "el dominio apunta a una dirección no pública; solicita la aprobación manual de NEXUS":
    "The domain points to a non-public address; ask NEXUS for manual approval",
  "no se pudo resolver el dominio": "The domain could not be resolved",
  "el archivo existe pero no contiene el token esperado": "The file exists but does not contain the expected token",
  "el registro TXT existe pero no contiene el token esperado":
    "The TXT record exists but does not contain the expected token",
  "demasiados objetivos en una solicitud": "Too many targets in one request",
  "objetivo no válido para esta solicitud": "Target not valid for this request",
  "no se pudo leer la solicitud": "The request could not be read",
  "no se pudo leer la autorización": "The authorization could not be read",
  "debes aceptar las reglas de enfrentamiento": "You must accept the rules of engagement",
  "los términos han cambiado; revísalos de nuevo": "The terms have changed; please review them again",
  "la evaluación ya está en marcha": "The assessment is already under way",
  "indica el nombre y el cargo de quien autoriza": "Enter the name and job title of the person authorizing",
  "indica un contacto de emergencia": "Enter an emergency contact",
  "las exclusiones deben tener como máximo 2000 caracteres": "Exclusions must be at most 2000 characters",
  "ventana de pruebas inválida (desde hoy, máximo 90 días)": "Invalid testing window (from today, at most 90 days)",
  "la solicitud no tiene objetivos": "The request has no targets",
  "no se pudo registrar la aceptación": "The acceptance could not be recorded",
  "las reglas de enfrentamiento ya están aceptadas": "The rules of engagement are already accepted",
  // org security policy
  "sesión expirada por la política de la organización; inicia sesión de nuevo":
    "Session expired under your organization's policy; please sign in again",
  "acceso no permitido desde esta red por la política de la organización":
    "Access from this network is not allowed by your organization's policy",
  "la política de la organización exige la verificación en dos pasos":
    "Your organization's policy requires two-step verification",
  "activa tu propia verificación en dos pasos antes de exigirla a la organización":
    "Turn on your own two-step verification before requiring it for the organization",
  "no se pudo leer la política": "The policy could not be read",
  "no se pudo guardar la política": "The policy could not be saved",
  // single sign-on
  "no hay inicio de sesión único configurado para ese dominio": "No single sign-on is configured for that domain",
  "el proveedor de identidad no responde": "The identity provider is not responding",
  "inicio de sesión caducado o no válido; vuelve a intentarlo": "Sign-in expired or invalid; please try again",
  "el inicio de sesión único no está activo para esta organización":
    "Single sign-on is not active for this organization",
  "no se pudo validar tu identidad con el proveedor": "Your identity could not be validated with the provider",
  "no tienes cuenta en esta organización; pide una invitación a un administrador":
    "You have no account in this organization; ask an administrator for an invitation",
  "esta cuenta pertenece a otra organización": "This account belongs to another organization",
  "la identidad del proveedor no coincide con la vinculada a esta cuenta":
    "The provider identity does not match the one linked to this account",
  "emisor (issuer) inválido: debe ser una URL https": "Invalid issuer: it must be an https URL",
  "client_id o client_secret inválido": "Invalid client_id or client_secret",
  "el rol por defecto debe ser member o auditor": "The default role must be member or auditor",
  "activa el SSO antes de exigirlo": "Enable SSO before requiring it",
  "ese dominio ya está configurado para SSO en otra organización":
    "That domain is already configured for SSO in another organization",
  "no se pudo leer la configuración SSO": "The SSO configuration could not be read",
  "no se pudo guardar la configuración SSO": "The SSO configuration could not be saved",
  "no se pudo eliminar la configuración SSO": "The SSO configuration could not be removed",
  "no se pudo cifrar el secreto": "The secret could not be encrypted",
};

// Messages that embed a value (host, scheme, status code). Matched after the
// exact table; the captured values are carried over untranslated.
export const SERVER_ERROR_PATTERNS_EN: [RegExp, string][] = [
  [/^no se encontró el registro TXT (\S+)$/, "TXT record $1 not found"],
  [/^no se pudo conectar por (https?)$/, "Could not connect over $1"],
  [/^(https?) respondió (\d+) en (\S+)$/, "$1 answered $2 for $3"],
  [/^la duración de sesión debe estar entre 1 y (\d+) horas$/, "The session length must be between 1 and $1 hours"],
  [/^red inválida: (.+)$/, "Invalid network: $1"],
  [/^máximo (\d+) redes$/, "At most $1 networks"],
  [
    /^el dominio (\S+) no tiene la propiedad verificada por DNS en Objetivos$/,
    "The domain $1 has no DNS-verified ownership in Targets",
  ],
  [/^indica entre 1 y (\d+) dominios de email$/, "Enter between 1 and $1 email domains"],
  [
    /^no se pudo leer la configuración OIDC del proveedor: (.+)$/,
    "Could not read the provider's OIDC configuration: $1",
  ],
  [
    /^tu IP actual \((\S+)\) no está en la lista; añádela para no bloquear tu propio acceso$/,
    "Your current IP ($1) is not in the list; add it so you do not lock yourself out",
  ],
];

// Internal infrastructure messages that must never be shown verbatim to a client
// (they leak implementation details and are not in the client's language).
export const SERVER_ERRORS_INTERNAL = new Set<string>(["管理后台数据源(PostgreSQL)未连接"]);
