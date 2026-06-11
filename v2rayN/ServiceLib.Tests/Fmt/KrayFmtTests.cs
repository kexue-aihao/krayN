using AwesomeAssertions;
using ServiceLib.Handler.Fmt;
using Xunit;

namespace ServiceLib.Tests.Fmt;

public class KrayFmtTests
{
    [Fact]
    public void GetShareUriAndResolveConfig_Kray_ShouldRoundTripBasicFields()
    {
        var source = CoreConfig.CoreConfigTestFactory.CreateKrayNode();

        var uri = FmtHandler.GetShareUri(source);

        uri.Should().NotBeNullOrWhiteSpace();
        uri!.Should().StartWith(Global.KrayProtocolShare);

        var resolved = FmtHandler.ResolveConfig(uri, out var msg);

        resolved.Should().NotBeNull($"uri: {uri}, msg: {msg}");
        resolved!.ConfigType.Should().Be(EConfigType.Kray);
        resolved.CoreType.Should().Be(ECoreType.kray);
        resolved.Address.Should().Be(source.Address);
        resolved.Port.Should().Be(source.Port);
        resolved.Username.Should().Be(source.Username);
        resolved.Password.Should().Be(source.Password);
        resolved.PublicKey.Should().Be(source.PublicKey);
        resolved.Network.Should().Be(source.Network);
        resolved.GetProtocolExtra().KrayPaddingMin.Should().Be(8);
        resolved.GetProtocolExtra().KrayPaddingMax.Should().Be(32);
    }

    [Fact]
    public void ResolveJsonSubscription_ShouldParseKrayProfiles()
    {
        const string payload = """
        {
          "profiles": [
            {
              "name": "hk-01",
              "endpoint": "kray.example:8443",
              "client_id": "client-1",
              "client_secret": "secret-1",
              "server_public_key": "public-key-1",
              "transport": "http-upgrade",
              "server_name": "edge.example",
              "headers": {"Host":"edge.example"},
              "padding_min": 4,
              "padding_max": 12
            }
          ]
        }
        """;

        var items = KrayFmt.ResolveJsonSubscription(payload, "sub");

        items.Should().NotBeNull();
        items!.Should().ContainSingle();
        var item = items[0];
        item.ConfigType.Should().Be(EConfigType.Kray);
        item.CoreType.Should().Be(ECoreType.kray);
        item.Remarks.Should().Be("hk-01");
        item.Address.Should().Be("kray.example");
        item.Port.Should().Be(8443);
        item.Network.Should().Be("http-upgrade");
        item.GetProtocolExtra().KrayHeaders.Should().Contain("Host");
    }
}
