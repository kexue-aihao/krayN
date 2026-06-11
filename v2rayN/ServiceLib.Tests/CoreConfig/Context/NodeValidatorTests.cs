using AwesomeAssertions;
using Xunit;

namespace ServiceLib.Tests.CoreConfig.Context;

public class NodeValidatorTests
{
    [Fact]
    public void Validate_KlessWithValidKeyMaterial_ShouldSucceed()
    {
        var node = CoreConfigTestFactory.CreateKrayNode();

        var result = NodeValidator.Validate(node, ECoreType.kray);

        result.Success.Should().BeTrue(string.Join(Environment.NewLine, result.Errors));
    }

    [Fact]
    public void Validate_KlessWithMixedClientSecretAndSigningKey_ShouldFail()
    {
        var node = CoreConfigTestFactory.CreateKrayNode();
        node.Password = node.PublicKey;

        var result = NodeValidator.Validate(node, ECoreType.kray);

        result.Success.Should().BeFalse();
        result.Errors.Should().Contain(e => e.Contains("must not be the same value"));
    }
}
