import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DeviceReportHistoryComponent } from './device-report-history.component';

describe('DeviceReportHistoryComponent', () => {
  let component: DeviceReportHistoryComponent;
  let fixture: ComponentFixture<DeviceReportHistoryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [DeviceReportHistoryComponent]
    });
    fixture = TestBed.createComponent(DeviceReportHistoryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
