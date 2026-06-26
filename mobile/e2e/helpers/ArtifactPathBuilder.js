/**
 * ArtifactPathBuilder — кастомный построитель путей для артефактов Detox.
 * Группирует артефакты по дате и имени теста.
 */
class ArtifactPathBuilder {
  constructor({ rootDir }) {
    this.rootDir = rootDir;
  }

  buildPathForTestInterceptionData({ testSummary }) {
    return `${this.rootDir}/interceptions/${this._buildFileName(testSummary)}.json`;
  }

  buildPathForArtifact({ testSummary, name }) {
    return `${this.rootDir}/${this._buildFileName(testSummary)}/${name}`;
  }

  _buildFileName(testSummary) {
    const { title, fullName, status } = testSummary;
    const sanitized = fullName.replace(/[^a-zA-Z0-9_-]/g, '_').slice(0, 100);
    return `${sanitized}_${status}`;
  }
}

module.exports = ArtifactPathBuilder;
