package syslog_acceptance_test

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var DeploymentName = func() string {
	return fmt.Sprintf("syslog-tests-%d", GinkgoParallelNode())
}

var BoshCmd = func(args ...string) *gexec.Session {
	boshArgs := []string{"-n", "-d", DeploymentName()}
	boshArgs = append(boshArgs, args...)
	boshCmd := exec.Command("bosh", boshArgs...)
	By("Performing command: bosh " + strings.Join(boshArgs, " "))
	session, err := gexec.Start(boshCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

var ForwarderSshCmd = func(command string) *gexec.Session {
	return BoshCmd("ssh", "forwarder", "-c", command)
}

var SendLogMessage = func(msg string) {
	session := ForwarderSshCmd(fmt.Sprintf("logger %s -t vcap.", msg))
	Eventually(session).Should(gexec.Exit(0))
}

var Cleanup = func() {
	BoshCmd("locks")
	session := BoshCmd("delete-deployment")
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
	Eventually(BoshCmd("locks")).ShouldNot(gbytes.Say(DeploymentName()))
}

var Deploy = func(manifest string) *gexec.Session {
	session := BoshCmd("deploy", manifest, "-v", fmt.Sprintf("deployment=%s", DeploymentName()))
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
	Eventually(BoshCmd("locks")).ShouldNot(gbytes.Say(DeploymentName()))
	return session
}

var ForwarderLog = func() *gexec.Session {
	// 47450 is CF's "enterprise ID" and uniquely identifies messages sent by our system
	session := BoshCmd("ssh", "storer", fmt.Sprintf("--command=%q", "cat /var/vcap/store/syslog_storer/syslog.log | grep '47450'"), "--json", "-r")
	Eventually(session).Should(gexec.Exit())
	return session
}

var AddFakeOldConfig = func() {
	By("Adding a file where the config used to live")
	session := ForwarderSshCmd("sudo bash -c 'echo fakeConfig=true > /etc/rsyslog.d/rsyslog.conf'")
	Eventually(session).Should(gexec.Exit(0))
}

var WriteToTestFile = func(message string) func() *gexec.Session {
	return func() *gexec.Session {
		session := ForwarderSshCmd(fmt.Sprintf("echo %s | sudo tee -a /var/vcap/sys/log/syslog_forwarder/file.log", message))
		Eventually(session).Should(gexec.Exit(0))
		return ForwarderLog()
	}
}

var DefaultLogfiles = func() *gexec.Session {
	session := BoshCmd("ssh", "forwarder", fmt.Sprintf("--command=%q", "sudo cat /var/log/{messages,syslog,user.log}"), "--json", "-r")
	Eventually(session).Should(gexec.Exit())
	return session
}