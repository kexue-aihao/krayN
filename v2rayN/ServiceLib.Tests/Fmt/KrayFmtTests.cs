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
        resolved!.ConfigType.Should().Be(EConfigType.KLESS);
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
              "client_secret": "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE",
              "server_public_key": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUY",
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
        item.ConfigType.Should().Be(EConfigType.KLESS);
        item.CoreType.Should().Be(ECoreType.kray);
        item.Remarks.Should().Be("hk-01");
        item.Address.Should().Be("kray.example");
        item.Port.Should().Be(8443);
        item.Network.Should().Be("http-upgrade");
        item.GetProtocolExtra().KrayHeaders.Should().Contain("Host");
    }

    [Fact]
    public void ResolveJsonSubscription_ShouldAcceptServerSigningKeyAlias()
    {
        const string payload = """
        {
          "profiles": [
            {
              "name": "jp-01",
              "endpoint": "kray.example:8443",
              "client_id": "client-1",
              "client_secret": "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE",
              "server_signing_key": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUY"
            }
          ]
        }
        """;

        var items = KrayFmt.ResolveJsonSubscription(payload, "sub");

        items.Should().NotBeNull();
        items![0].PublicKey.Should().Be("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUY");
    }

    [Fact]
    public void ResolveShareUri_ShouldAcceptServerSigningKeyAlias()
    {
        const string uri = "kray://client-1@kray.example:8443?client_secret=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE&server_signing_key=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUY#jp-01";

        var item = FmtHandler.ResolveConfig(uri, out var msg);

        item.Should().NotBeNull($"msg: {msg}");
        item!.PublicKey.Should().Be("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUY");
    }
}
