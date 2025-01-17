from aqueduct.models.integration import Integration, IntegrationInfo


class LambdaIntegration(Integration):
    """
    Class for K8s integration.
    """

    def __init__(self, metadata: IntegrationInfo):
        self._metadata = metadata

    def describe(self) -> None:
        """Prints out a human-readable description of the K8s integration."""
        print("==================== Lambda Integration  =============================")
        self._metadata.describe()
