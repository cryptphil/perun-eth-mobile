package network.perun.app;

import android.util.Log;

import prnm.*;

import java.math.*;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.jupiter.api.Assertions;

import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import androidx.test.runner.AndroidJUnit4;
import androidx.test.core.app.ApplicationProvider;

import static com.google.common.truth.Truth.assertThat;
import static org.awaitility.Awaitility.*;
import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.notNullValue;

@RunWith(AndroidJUnit4.class)
public class PrnmTest implements prnm.NewChannelCallback, prnm.ProposalHandler, prnm.UpdateHandler {
    Client client;
    Context ctx = Prnm.contextWithTimeout(120);
    String prefix;

    AtomicInteger receivedTx = new AtomicInteger(0);
    AtomicReference<PaymentChannel> ch = new AtomicReference<PaymentChannel>(null);

    @Test
    public void runAlice() throws Exception {
        run(Setup.ALICE);
    }

    @Test
    public void runBob() throws Exception {
        run(Setup.BOB);
    }

    public void run(Setup s) throws Exception {
        sharedSetup(s);
        if (s == Setup.ALICE)
            aliceTest(s);
        else
            bobTest(s);
        sharedTeardown(s);
    }

    public void sharedSetup(Setup setup) throws Exception {
        Prnm.setLogLevel(6);    // TRACE
        prefix = "prnm-" + setup.MyAlias;
        // Get the Apps data directory.
        String appDir = ApplicationProvider.getApplicationContext().getFilesDir().getAbsolutePath();
        String ksPath = appDir + "/keystore";
        String dbPath = appDir + "/database";

        String password = "0123456789";

        Wallet wallet = Prnm.newWallet(ksPath, password);
        Address onChain = wallet.importAccount(setup.MySK);

        String ethUrl = "ws://10.5.0.9:8545";
        Config cfg = new Config(setup.MyAlias, onChain, setup.Adjudicator, setup.Assetholder, ethUrl, "0.0.0.0", 5750);
        client = new Client(ctx, cfg, wallet);

        client.addPeer(setup.PeerAddress, setup.PeerHost, setup.PeerPort);
        client.onNewChannel(this);
        new Thread(() -> {
            client.handle(this, this);
        }).start();

        client.enablePersistence(dbPath);
        client.restore(ctx);
    }

    public void aliceTest(Setup setup) throws Exception {
        // Open Channel.
        Thread.sleep(5000);
        BigInts initBals = Prnm.newBalances(eth(20), eth(20));
        log("Opening Channel…");
        client.proposeChannel(ctx, setup.PeerAddress, 60, initBals);
        Thread.sleep(100);  // BUG/FEATURE in go-perun. Bob otherwise reports: 'received update for unknown channel'
        log("Channel opened.");
        // Send one tx to Bob and wait for his tx.
        for (int i = 0; i < 5; ++i) {
            log("Sending TX…");
            ch.get().send(ctx, eth(1));
            log("TX sent");
            await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
        }
        // Final balance check.
        BigInts newBals = ch.get().getState().getBalances();
        assertThat(newBals.get(0)).isEqualTo(eth(10));
        assertThat(newBals.get(1)).isEqualTo(eth(30));
    }

    public void bobTest(Setup setup) throws Exception {
        // Wait for the channel to be created.
        log("Waiting for channel…");
        await().atMost(20, TimeUnit.SECONDS).untilAtomic(ch, notNullValue());
        log("Channel opened.");
        // Wait for Alice to send a tx and then send one back.
        for (int i = 0; i < 5; ++i) {
            await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
            log("Sending TX…");
            ch.get().send(ctx, eth(2));
            log("TX sent.");
        }
        log("10 TX received.");
    }

    public void sharedTeardown(Setup setup) throws Exception {
        if (setup == Setup.BOB) {
            // Wait for Alice to not mess up here receivedTX counter with a final update.
            Thread.sleep(1000);
            log("Finalizing");
            ch.get().finalize(ctx);
        }
        Thread.sleep(5000);     // Wait for Bob register
        log("Settling");
        ch.get().settle(ctx);
        log("Done");
    }

    @Override
    public void onNew(PaymentChannel channel) {
        log("onNewChannel");
        ch.set(channel);
        new Thread(() -> {
            Assertions.assertDoesNotThrow(() -> {
                log("Watcher started");
                channel.watch();
                log("Watcher stopped");
            });
        }).start();
        log("onNewChannel done");
    }

    @Override
    public void handleProposal(ChannelProposal proposal, ProposalResponder responder) {
        Assertions.assertDoesNotThrow(() -> {
            BigInts bals = proposal.getInitBals();
            log("handleProposal Bals: " + bals.get(0).toString() + "/" + bals.get(1).toString() + " CD: " + proposal.getChallengeDuration() + "s");
            responder.accept(ctx);
            log("handleProposal done");
        });
    }

    @Override
    public void handleUpdate(ChannelUpdate update, UpdateResponder responder) {
        Assertions.assertDoesNotThrow(() -> {
            log("handleUpdate");
            responder.accept(ctx);
            log("handleUpdate done #" +receivedTx.incrementAndGet());
        });
    }

    static BigInt eth(int i) {
        return Prnm.newBigIntFromBytes(BigInteger.valueOf(i).multiply(new BigInteger("1000000000000000000")).toByteArray());
    }

    public void log(String msg) {
        Log.i(prefix, msg);
    }
}

class Setup {
    public Address Adjudicator, Assetholder;
    public String MyAlias, MySK;
    public Address PeerAddress;
    public String PeerHost;
    public int PeerPort;

    private Setup(String myAlias, String mySK, Address peerAddress, String peerHost, int peerPort, Address adj, Address asset) {
        Adjudicator = adj;
        Assetholder = asset;
        MyAlias = myAlias;
        MySK = mySK;
        PeerAddress = peerAddress;
        PeerHost = peerHost;
        PeerPort = peerPort;
    }

    public final static Setup ALICE = new Setup(
            "Alice", "0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9",
            new Address("0xA298Fc05bccff341f340a11FffA30567a00e651f"), "10.5.0.6", 5750,
            new Address("0x94503e14e26a433c0802e04f2ac1bb1ce77321f5"), new Address("0xc2f95e626123a61bed88752475b870efc4a5f453"));

    public final static Setup BOB = new Setup(
            "Bob", "0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a",
            new Address("0x05e71027e7d3bd6261de7634cf50F0e2142067C4"), "10.5.0.6", 5753,
            null, null);
}