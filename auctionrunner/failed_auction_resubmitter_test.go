package auctionrunner_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResubmitFailedAuctions", func() {
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider
	var results auctiontypes.AuctionResults
	var maxRetries int

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
		maxRetries = 3
	})

	It("always returns succesful work untouched", func() {
		results = auctiontypes.AuctionResults{
			SuccessfulLRPStarts: []auctiontypes.LRPStartAuction{
				BuildStartAuction(BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10), timeProvider.Now()),
				BuildStartAuction(BuildLRPStartAuction("pg-2", 1, "lucid64", 10, 10), timeProvider.Now()),
			},
			SuccessfulTasks: []auctiontypes.TaskAuction{
				BuildTaskAuction(BuildTask("tg-1", "lucid64", 10, 10), timeProvider.Now()),
				BuildTaskAuction(BuildTask("tg-2", "lucid64", 10, 10), timeProvider.Now()),
			},
			FailedLRPStarts: []auctiontypes.LRPStartAuction{},
			FailedTasks:     []auctiontypes.TaskAuction{},
		}

		out := ResubmitFailedAuctions(batch, results, maxRetries)
		Ω(out).Should(Equal(results))
	})

	It("should not resubmit if there is nothing to resubmit", func() {
		ResubmitFailedAuctions(batch, auctiontypes.AuctionResults{}, maxRetries)
		Ω(batch.HasWork).ShouldNot(Receive())
	})

	Context("if there is failed work", func() {
		var retryableStartAuction, failedStartAuction auctiontypes.LRPStartAuction
		var retryableTaskAuction, failedTaskAuction auctiontypes.TaskAuction

		BeforeEach(func() {
			retryableStartAuction = BuildStartAuction(BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10), timeProvider.Now())
			retryableStartAuction.Attempts = maxRetries
			failedStartAuction = BuildStartAuction(BuildLRPStartAuction("pg-2", 1, "lucid64", 10, 10), timeProvider.Now())
			failedStartAuction.Attempts = maxRetries + 1

			retryableTaskAuction = BuildTaskAuction(BuildTask("tg-1", "lucid64", 10, 10), timeProvider.Now())
			retryableTaskAuction.Attempts = maxRetries
			failedTaskAuction = BuildTaskAuction(BuildTask("tg-2", "lucid64", 10, 10), timeProvider.Now())
			failedTaskAuction.Attempts = maxRetries + 1

			results = auctiontypes.AuctionResults{
				FailedLRPStarts: []auctiontypes.LRPStartAuction{retryableStartAuction, failedStartAuction},
				FailedTasks:     []auctiontypes.TaskAuction{retryableTaskAuction, failedTaskAuction},
			}
		})

		It("should resubmit work that can be retried and does not return it, but returns work that has exceeded maxretries without resubmitting it", func() {
			out := ResubmitFailedAuctions(batch, results, maxRetries)
			Ω(out.FailedLRPStarts).Should(ConsistOf(failedStartAuction))
			Ω(out.FailedTasks).Should(ConsistOf(failedTaskAuction))

			resubmittedStarts, resubmittedTasks := batch.DedupeAndDrain()
			Ω(resubmittedStarts).Should(ConsistOf(retryableStartAuction))
			Ω(resubmittedTasks).Should(ConsistOf(retryableTaskAuction))
		})
	})
})
