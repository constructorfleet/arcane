import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from release import main, metadata, release_body, validate


class ReleaseContractTests(unittest.TestCase):
    def setUp(self):
        self.previous = metadata('REGISTRY=registry.example\nREPOSITORY=constructorfleet/arcane\nVERSION=3.0.1-cf.3\nPLATFORMS=linux/amd64\nHARBOR_PROJECT_ID=2')
        self.current = {**self.previous, 'VERSION': '3.0.1-cf.4'}
        self.notes = '# 3.0.1-cf.4\n\n## Changes\n\n- Release workflow.\n'

    def test_next_version_and_notes(self):
        validate(self.current, self.previous, self.notes)

    def test_reused_and_skipped_versions_are_rejected(self):
        for version in ('3.0.1-cf.3', '3.0.1-cf.5', '3.0.0-cf.4'):
            with self.subTest(version=version), self.assertRaises(ValueError):
                validate({**self.current, 'VERSION': version}, self.previous, '# ' + version + '\n\nChanges\n- Test')

    def test_new_upstream_base_starts_at_one(self):
        validate({**self.current, 'VERSION': '3.1.0-cf.1'}, self.previous, '# 3.1.0-cf.1\n\nChanges\n- Rebase')

    def test_mismatched_notes_are_rejected(self):
        with self.assertRaises(ValueError):
            validate(self.current, self.previous, self.notes.replace('cf.4', 'cf.3'))

    def test_published_retry_is_marked_for_skipping_builds(self):
        sha = 'b' * 40
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            with patch.dict(os.environ, {'GITEA_SHA': sha, 'GITHUB_OUTPUT': str(output)}), \
                    patch('sys.argv', ['release.py', 'metadata']), \
                    patch('release.api', side_effect=[{'commit': {'sha': sha}}, {'target_commitish': sha, 'draft': False}]) as request:
                main()
            self.assertIn('published=true', output.read_text())
            self.assertTrue(all(call.args[0] == 'GET' for call in request.call_args_list))

    def test_release_includes_both_immutable_images_and_links(self):
        digest = 'sha256:' + 'a' * 64
        body = release_body(self.current, self.notes, 'b' * 40, {'manager': digest, 'agent': digest},
                            {'number': 7, 'html_url': 'https://gitea.example/owner/repo/pulls/7'}, 'https://gitea.example', 'owner/repo')
        for image in ('manager', 'agent'):
            self.assertIn(f'arcane-{image}@{digest}', body)
            self.assertIn(f'arcane-{image}:3.0.1-cf.4', body)
            self.assertIn(f'/harbor/projects/2/repositories/arcane-{image}/artifacts-tab', body)
            self.assertIn(f'/api/v2.0/projects/constructorfleet/repositories/arcane-{image}/artifacts/{digest}', body)
        self.assertIn('/pulls/7', body)
        self.assertIn('/commit/' + 'b' * 40, body)
        with self.assertRaises(ValueError):
            release_body(self.current, self.notes, 'b' * 40, {'manager': 'latest'}, None, 'https://gitea.example', 'owner/repo')


if __name__ == '__main__':
    unittest.main()
