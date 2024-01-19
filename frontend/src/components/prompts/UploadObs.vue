<template>
  <div class="card floating">
    <div class="card-content">
      <p v-if="selectedCount === 1">
        {{ $t("prompts.deleteMessageSingle") }}
      </p>
      <p v-else>
        {{ $t("prompts.deleteMessageMultiple", { count: selectedCount }) }}
      </p>
    </div>
    <div class="card-action">
      <button
        @click="$store.commit('closeHovers')"
        class="button button--flat button--grey"
        :aria-label="$t('buttons.cancel')"
        :title="$t('buttons.cancel')"
      >
        {{ $t("buttons.cancel") }}
      </button>
      <button
        @click="submit"
        class="button button--flat button--red"
        :aria-label="$t('buttons.upload')"
        :title="$t('buttons.upload')"
      >
        {{ $t("buttons.upload") }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapGetters, mapMutations, mapState } from "vuex";
import { files as api } from "@/api";
import buttons from "@/utils/buttons";

export default {
  name: "upload-obs",
  computed: {
    ...mapGetters(["isListing", "selectedCount", "currentPrompt"]),
    ...mapState(["req", "selected"]),
  },
  methods: {
    ...mapMutations(["closeHovers"]),
    submit: async function () {
      buttons.loading("upload-obs");

      try {
        if (!this.isListing) {
          await api.uploadToObs(this.req.items[this.selected[0]].url);
          buttons.success("upload-obs");

          this.currentPrompt?.confirm();
          this.closeHovers();
          return;
        }

        this.closeHovers();

        let files = [];

        if (this.selectedCount > 0) {
          for (let i of this.selected) {
            files.push(this.req.items[i].url);
          }
        } else {
          files.push(this.$route.path);
        }

        await api.uploadToObs(...files);
        buttons.success("upload-obs");
        this.$store.commit("setReload", true);
      } catch (e) {
        buttons.done("upload-obs");
        this.$showError(e);
        if (this.isListing) this.$store.commit("setReload", true);
      }
    },
  },
};
</script>
