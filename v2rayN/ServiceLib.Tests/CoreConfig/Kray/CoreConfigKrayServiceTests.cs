using AwesomeAssertions;
using ServiceLib.Services.CoreConfig;
using Xunit;

namespace ServiceLib.Tests.CoreConfig.Kray;

public class CoreConfigKrayServiceTests
{
    [Fact]
    public void GenerateClientConfigContent_ShouldGenerateNativeKrayConfig()
    {
        var config = CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        CoreConfigTestFactory.BindAppManagerConfig(config);
        var node = CoreConfigTestFactory.CreateKrayNode();
        var context = CoreConfigTestFactory.CreateContext(config, node, ECoreType.kray);

        var result = new CoreConfigKrayService(context).GenerateClientConfigContent();

        result.Success.Should().BeTrue($"ret msg: {result.Msg}");
        result.Data.Should().NotBeNull();

        var root = JsonUtils.ParseJson(result.Data!.ToString())!.AsObject();
        root["auto_start"]!.GetValue<bool>().Should().BeTrue();
        root["active_profile_id"]!.GetValue<string>().Should().Be(node.IndexId);
        root["local"]!["socks_address"]!.GetValue<string>().Should().Be("127.0.0.1:10808");

        var profile = root["profiles"]!.AsArray()[0]!.AsObject();
        profile["transport"]!.GetValue<string>().Should().Be("websocket");
        profile["endpoint"]!.GetValue<string>().Should().Be("kray.example:8443");
        profile["client_id"]!.GetValue<string>().Should().Be(node.Username);
        profile["client_secret"]!.GetValue<string>().Should().Be(node.Password);
        profile["server_public_key"]!.GetValue<string>().Should().Be(node.PublicKey);
        profile["headers"]!["Host"]!.GetValue<string>().Should().Be("edge.example");
        profile["handshake_padding"]!["min"]!.GetValue<int>().Should().Be(8);
        profile["handshake_padding"]!["max"]!.GetValue<int>().Should().Be(32);
    }
}
