import { ComponentFixture, TestBed } from '@angular/core/testing';

import { HomeSummaryReportComponentComponent } from './home-summary-report-component.component';

describe('HomeSummaryReportComponentComponent', () => {
  let component: HomeSummaryReportComponentComponent;
  let fixture: ComponentFixture<HomeSummaryReportComponentComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [HomeSummaryReportComponentComponent]
    });
    fixture = TestBed.createComponent(HomeSummaryReportComponentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
