# Transporte real Phantom

```sh
nexus phantom up
nexus phantom status
nexus phantom check --fault
nexus phantom vpn-test
```

`up` compila el binario Go y construye la imagen Debian con Tor y wireguard-tools.
Necesita Docker, Go y conexión para descargar paquetes. El comando equivalente
 dentro de la terminal interactiva es `/phantom up`.

El proxy escucha solo en `127.0.0.1:1080`. Configura el cliente como
`socks5h://127.0.0.1:1080` para enviar el nombre de destino sin resolverlo localmente.
Solo admite TCP CONNECT; UDP y direcciones IP numéricas se rechazan. La identidad
 de aislamiento SOCKS de Tor cambia al reiniciar el proxy. No constituye una
identidad de navegador aislada ni impide que una cuenta iniciada identifique al usuario.

El contenedor `proxy` solo pertenece a una red Docker interna. `tor` participa en
esa red y en su propia red de salida. No se publican los puertos SOCKS internos
 de Tor ni un puerto de control. El proxy tiene límites de conexiones y tiempos
 de espera, conserva la semiconexión TCP y rechaza transportes ausentes o fallidos.
No añade bytes de relleno al protocolo de aplicación.

## VPN con servidor propio o proveedor

```sh
chmod 600 /ruta/privada/cliente.conf
nexus phantom up --vpn-profile /ruta/privada/cliente.conf
nexus phantom check --fault
```

El perfil debe pertenecer al usuario. Se acepta un solo peer, endpoint IPv4
literal, dirección IPv4 del túnel, resolver DNS y `AllowedIPs = 0.0.0.0/0`.
Los hooks, comandos, peers múltiples y rutas parciales se rechazan. Un perfil
con dirección IPv6 adicional puede importarse, pero IPv6 queda bloqueado.
Las claves nunca aparecen en los informes ni se copian al repositorio.

En este modo Tor comparte el espacio de red de WireGuard, con firewall DROP.
Fuera de `wg0` solo se permite el UDP hacia el endpoint configurado y las respuestas
SOCKS a clientes admitidos. El resolver se monta explícitamente; se bloquea el DNS
implícito de Docker. Tor espera un handshake WireGuard reciente para arrancar.
Detener el túnel conserva el bloqueo. Solo la pasarela obtiene NET_ADMIN y permiso
para leer el perfil privado; Tor y el proxy carecen de esas capacidades.

Esta configuración opera en contenedores y no cambia las rutas ni el firewall del
host. No es una VPN global de Windows. `down` detiene los contenedores de Phantom
sin borrar el estado persistente de Tor. Tras activar VPN, posteriores `up`
conservan ese modo: nunca se desactiva automáticamente para recuperar conectividad.

## Verificación y límites

`check --fault` verifica Tor, salidas directas bloqueadas y recuperación después de
un corte. Interrumpe brevemente solo Tor. `vpn-test` crea y elimina dos contenedores
temporales; mide tráfico HTTP/DNS real por WireGuard y bloqueo al eliminar la interfaz.
El test local no prueba un proveedor ni una VPN remota a Internet.

La biblioteca Onion/Mixnet original no queda desplegada por estos comandos. No se
 afirma anonimato absoluto ni cobertura automática del backend o del sistema operativo.

Referencias de implementación: [DNS remoto en Tor](https://support.torproject.org/little-t-tor/troubleshooting/check-for-leaks/),
[SOCKS en Tor](https://spec.torproject.org/socks-extensions.html),
[WireGuard y espacios de red](https://www.wireguard.com/netns/).
