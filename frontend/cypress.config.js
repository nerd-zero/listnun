const { defineConfig } = require('cypress');

module.exports = defineConfig({
  env: {
    apiUrl: 'http://localhost:9000',
    serverInitCmd:
      'pkill -9 listnun; cd ../ && LISTNUN_ADMIN_USER=admin LISTNUN_ADMIN_PASSWORD=listnun123 ./listnun --install --yes && setsid ./listnun </dev/null >/dev/null 2>&1 &',
    serverInitBlankCmd:
      'pkill -9 listnun; cd ../ && ./listnun --install --yes && setsid ./listnun </dev/null >/dev/null 2>&1 &',
    LISTNUN_ADMIN_USER: 'admin',
    LISTNUN_ADMIN_PASSWORD: 'listnun123',
  },
  viewportWidth: 1400,
  viewportHeight: 950,
  e2e: {
    experimentalRunAllSpecs: true,
    testIsolation: false,
    experimentalSessionAndOrigin: false,
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
    setupNodeEvents(on, config) {
      return require('./cypress/plugins/index.js')(on, config);
    },
    baseUrl: 'http://localhost:9000',
  },
});
