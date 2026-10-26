import pathlib
import tempfile
import unittest

from release_smoke import lab_dsn


class ReleaseBoundaryTests(unittest.TestCase):
    def test_only_local_dedicated_database_is_accepted(self):
        for dsn in ["postgres://tester:fixture@127.0.0.1:5432/nexus_lab_release?sslmode=disable", "postgresql://tester:fixture@[::1]:5432/nexus_lab_release"]:
            self.assertEqual(lab_dsn(dsn), dsn)

    def test_unix_socket_must_stay_in_temporary_lab_directory(self):
        with tempfile.TemporaryDirectory(prefix="nexus-smoke-dsn-") as directory:
            dsn = "postgres://tester:fixture@/nexus_lab_release?host=" + directory
            self.assertEqual(lab_dsn(dsn), dsn)
        with self.assertRaises((OSError, ValueError)):
            lab_dsn("postgres://tester@/nexus_lab_release?host=/var/run/postgresql")

    def test_public_databases_and_ambiguous_options_are_rejected(self):
        for dsn in ["", "postgres://tester@127.0.0.1/production", "postgres://tester@example.com/nexus_lab_release", "postgres://tester@10.0.0.1/nexus_lab_release", "postgres://tester@127.0.0.1/nexus_lab_release?host=/tmp", "postgres://tester@127.0.0.1/nexus_lab_release?options=unsafe", "postgres://tester@127.0.0.1/nexus_lab_release?sslmode=disable&sslmode=require"]:
            with self.subTest(dsn=dsn), self.assertRaises(ValueError):
                lab_dsn(dsn)


if __name__ == "__main__":
    unittest.main()
