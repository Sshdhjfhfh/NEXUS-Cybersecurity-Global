# Phantom: estado comprobado

La auditoría inicial detectó salida directa por defecto y componentes Onion/VPN
que solo procesaban datos en memoria. Esta entrega añade un transporte operativo
separado en `phantom/runtime`, basado en Tor y WireGuard estándar.

## Resultados reales

- El proxy rechaza el arranque si falta un transporte. Nunca retrocede a una
  conexión directa cuando el transporte falla.
- La red pública Tor completó el arranque. `check.torproject.org/api/ip`, consultado
  por el proxy, reconoció la IP de salida como Tor.
- Desde el contenedor del proxy fallaron las conexiones directas IPv4/IPv6 y la
  resolución local del nombre de destino. Los nombres se envían a Tor.
- Detener Tor bloqueó el proxy; reiniciarlo recuperó una salida reconocida como Tor.
- Dos contenedores con WireGuard del núcleo intercambiaron tráfico HTTP y DNS,
  con handshake autenticado y contadores de transferencia en ambas direcciones.
- El acceso TCP directo por la red base estuvo bloqueado. Al eliminar `wg0`,
  las nuevas conexiones y el DNS permanecieron bloqueados. IPv6 tiene política DROP.

Evidencias regenerables: `phantom/runtime/artifacts/network-check.json` y
`phantom/runtime/artifacts/vpn-check.json`. Los resultados describen el momento
de ejecución, no una garantía permanente.

## Qué sigue pendiente

- La salida VPN a Internet requiere un perfil y un servidor remoto del usuario.
- El transporte operativo usa Tor; `phantom/onion` y `phantom/mixnet` propios
  siguen siendo componentes experimentales y no componen una red desplegada.
- El proxy protege únicamente las conexiones que se le envían. No cubre por sí
  solo todo Windows, WSL, el backend de Nexus ni programas que ignoren el proxy.
- No se certifican anonimato absoluto, resistencia a correlación de tráfico,
  diez capas desplegadas ni criptografía poscuántica del túnel WireGuard estándar.

Comandos: `nexus phantom up`, `nexus phantom check --fault`,
`nexus phantom vpn-test`, `nexus phantom audit`. Detalles en
[`phantom/runtime/README.md`](../phantom/runtime/README.md).
