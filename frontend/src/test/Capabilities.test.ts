import { describe, expect, it } from 'vitest';
import { resolveCapability } from '../capabilities';

describe('capability gates', () => {
  it('aceita capabilities canônicas suportadas', () => {
    const gate = resolveCapability([{
      id: 'samba.testparm',
      feature: 'testparm',
      state: 'suportado',
      scope: 'Samba',
      evidence: 'Binario detectado.'
    }], 'samba.testparm');

    expect(gate.available).toBe(true);
  });

  it('mantém capability não verificada indisponível e explica o motivo', () => {
    const gate = resolveCapability([{
      id: 'samba.ad_dc',
      feature: 'AD DC',
      state: 'nao_verificado',
      scope: 'Samba',
      evidence: 'Homologacao pendente.'
    }], 'samba.ad_dc');

    expect(gate.available).toBe(false);
    expect(gate.reason).toContain('Homologacao pendente');
  });
});
