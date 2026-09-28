import { defineConfig } from 'orval';

export default defineConfig({
  listnun: {
    input: {
      target: '../docs/swagger.yaml',
    },
    output: {
      mode: 'tags-split',
      target: 'src/api/generated/endpoints',
      schemas: 'src/api/generated/model',
      client: 'axios',
      formatter: 'prettier',
      override: {
        mutator: {
          path: 'src/api/mutator.ts',
          name: 'httpMutator',
        },
      },
    },
  },
});
