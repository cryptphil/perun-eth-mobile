package network.perun.app;

import prnm.*;

import java.math.*;
import java.nio.ByteBuffer;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import android.util.Log;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.jupiter.api.Assertions;
import androidx.test.runner.AndroidJUnit4;
import androidx.test.core.app.ApplicationProvider;
import static com.google.common.truth.Truth.assertThat;
import static org.awaitility.Awaitility.*;
import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.notNullValue;

@RunWith(AndroidJUnit4.class)
public class PrnmTest implements prnm.NewChannelCallback, prnm.ProposalHandler, prnm.UpdateHandler, prnm.ConcludedEventHandler {

    /**
     * This class contains four entry points for the test-runner; two per peer and test case.
     *
     * runFirst* sets up a channel, sends TXs and persists it.
     * runSecond* asserts that the channel can be restored from persistence.
     */

    Client client;
    Context ctx = Prnm.contextWithTimeout(120);
    String prefix;
    Setup setup;

    AtomicInteger receivedTx = new AtomicInteger(0);
    AtomicBoolean watcherStopped = new AtomicBoolean(false);
    AtomicReference<PaymentChannel> ch = new AtomicReference<PaymentChannel>(null);

    @Test
    /**
     * Alice entry point for the first test.
     */
    public void runFirstTestAlice() throws Exception {
        runFirstTest(Setup.ALICE);
    }

    @Test
    /**
     * Bob entry point for the test runner to run the first test.
     */
    public void runFirstTestBob() throws Exception {
        runFirstTest(Setup.BOB);
    }

    @Test
    /**
     * Alice entry point for the test runner to run the second test.
     */
    public void runSecondTestAlice() throws Exception {
        runSecondTest(Setup.ALICE);
    }

    @Test
    /**
     * Bob entry point for the test runner to run the second test.
     */
    public void runSecondTestBob() throws Exception {
        runSecondTest(Setup.BOB);
    }

    /**
     * This function does the following:
     *  both:
     *      - sharedSetup
     *      - enables persistence
     *  Alice:
     *      - proposes a payment channel to Bob
     *  Bob:
     *      - accepts the proposed channel from Alice
     *  both (repeated 3 times):
     *      - send 1 TX and wait for a TX from the other peer, Alice goes first
     *      - exit without closing the channel
     * @param s
     * @throws Exception
     */
    public void runFirstTest(Setup s) throws Exception {
        if (s == Setup.ALICE) {
            sharedSetup(s);
            proposeChannel(s);
            sendTx(false, 3);
        }
        else {
            // Set the contracts to null so Bob deploys them.
            s.Adjudicator = null;
            s.Assetholder = null;
            sharedSetup(s);
            acceptChannel(s);
            sendTx(true, 5);
        }
    }

    /**
     * This function does the following:
     *  both:
     *      - sharedSetup
     *      - restores the channel with persistence
     *  both (repeated 3 times):
     *      - send 1 TX and wait for a TX from the other peer, Bob goes first
     *  Alice:
     *      - asserts the final balances
     *  Bob:
     *      - finalizes the channel
     *  both:
     *      - settle the channel
     * @param s
     * @throws Exception
     */
    public void runSecondTest(Setup s) throws Exception {
        sharedSetup(s);

        // Test that the persistence restored the channel.
        BigInts bals = ch.get().getState().getBalances();
        log("assert Bals: " + bals.get(0).toString() + "/" + bals.get(1).toString());
        assertThat(bals.get(0).string()).isEqualTo(eth(106).string());
        assertThat(bals.get(1).string()).isEqualTo(eth(94).string());

        if (s == Setup.ALICE)
            sendTx(true, 1);
        else
            sendTx(false, 2);
        sharedTeardown(s);
    }

    /**
     * This function:
     * - imports an account from the Setup
     * - connects to the Ethereum node
     * - creates a prnm client
     * - adds the other peer to the client
     * - starts the proposal, update and newChannel handlers
     */
    public void sharedSetup(Setup setup) throws Exception {
        this.setup = setup;
        Prnm.setLogLevel(6);    // TRACE
        prefix = "prnm-" + setup.MyAlias;   // android log prefix
        String appDir = ApplicationProvider.getApplicationContext().getFilesDir().getAbsolutePath();
        String ksPath = appDir + "/keystore";
        String dbPath = appDir + "/database";

        Wallet wallet = Prnm.newWallet(ksPath, "0123456789");
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

    public void proposeChannel(Setup setup) throws Exception {
        Thread.sleep(5000);
        BigInts initBals = Prnm.newBalances(eth(100), eth(100));
        log("Opening Channel…");
        client.proposeChannel(ctx, setup.PeerAddress, 60, initBals);
        Thread.sleep(100);  // BUG in go-perun. Bob otherwise reports: 'received update for unknown channel'
        log("Channel opened.");
    }

    public void acceptChannel(Setup setup) throws Exception {
        log("Waiting for channel…");
        await().atMost(20, TimeUnit.SECONDS).untilAtomic(ch, notNullValue());
        log("Channel opened.");
    }

    /**
     *
     * @param wait Decides whether this peer waits first or sends an update first.
     * @throws Exception
     */
    public void sendTx(boolean wait, int amount) throws Exception {
        for (int i = 0; i < 3; ++i) {
            if (wait)
                await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
            log("Sending TX…");
            ch.get().send(ctx, eth(amount));
            log("TX sent.");
            if (!wait)
                await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
        }

    }

    public void sharedTeardown(Setup setup) throws Exception {
        if (setup == Setup.BOB) {
            Thread.sleep(1000);
            log("Finalizing");
            ch.get().finalize(ctx);
        } else {
            Thread.sleep(5000);
            log("Settling");
            ch.get().settle(ctx, false);
        }

        await().atMost(20, TimeUnit.SECONDS).untilAtomic(watcherStopped, is(true));
        log("Done");
        // Final balance check.
        BigInts newBals = ch.get().getState().getBalances();
        assertThat(newBals.get(0).string()).isEqualTo(eth(109).string());
        assertThat(newBals.get(1).string()).isEqualTo(eth(91).string());
    }

    @Override
    public void onNew(PaymentChannel channel) {
        log("onNewChannel");
        ch.set(channel);
        new Thread(() -> {
            Assertions.assertDoesNotThrow(() -> {
                log("Watcher started");
                channel.watch(this);
                log("Watcher stopped");
                watcherStopped.set(true);
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

    @Override
    public void handleConcluded(byte[] id) {
        if (this.setup == Setup.BOB) {
            log("handleConcluded");
            Assertions.assertDoesNotThrow(() -> {
                ch.get().settle(ctx, true);
                log("handleConcluded: Settled");
            });
        } else {
            log("handleConcluded: skipped");
        }
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
            new Address("0x94503e14e26a433c0802e04f2ac1bb1ce77321f5"), new Address("0xc2f95e626123a61bed88752475b870efc4a5f453"));
}
