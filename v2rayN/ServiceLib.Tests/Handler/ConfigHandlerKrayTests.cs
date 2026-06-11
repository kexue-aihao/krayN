using AwesomeAssertions;
using ServiceLib.Handler;
using ServiceLib.Helper;
using Xunit;

namespace ServiceLib.Tests.Handler;

public class ConfigHandlerKrayTests
{
    [Fact]
    public async Task AddBatchServers_KrayJsonSubscription_ShouldStoreKrayProfiles()
    {
        var config = CoreConfig.CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        CoreConfig.CoreConfigTestFactory.BindAppManagerConfig(config);
        SQLiteHelper.Instance.CreateTable<ProfileItem>();
        SQLiteHelper.Instance.CreateTable<SubItem>();

        var sub = new SubItem
        {
            Id = $"sub-{Guid.NewGuid():N}",
            Remarks = "Kray Subscription",
            Url = "https://example.invalid/sub"
        };
        await SQLiteHelper.Instance.InsertAsync(sub);

        const string payload = """
        {
          "profiles": [
            {
              "name": "hk-01",
              "endpoint": "kray.example:8443",
              "client_id": "client-1",
              "client_secret": "secret-1",
              "server_public_key": "public-key-1",
              "transport": "websocket",
              "server_name": "edge.example"
            }
          ]
        }
        """;

        var count = await ConfigHandler.AddBatchServers(config, payload, sub.Id, true);

        count.Should().Be(1);
        var profiles = await SQLiteHelper.Instance.TableAsync<ProfileItem>().Where(t => t.Subid == sub.Id).ToListAsync();
        profiles.Should().ContainSingle();
        profiles[0].ConfigType.Should().Be(EConfigType.Kray);
        profiles[0].CoreType.Should().Be(ECoreType.kray);
        profiles[0].Address.Should().Be("kray.example");
        profiles[0].Network.Should().Be("websocket");
        profiles[0].IsValid().Should().BeTrue();
    }
}
